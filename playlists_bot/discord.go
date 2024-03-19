package main

import (
	"context"
	"strings"
	"time"

	"database/sql"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

type messageComponents []discordgo.MessageComponent

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "add_playlist",
			Description: "Command for adding a playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "id",
					Description: "playlist string",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "title",
					Description: "playlist title",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "description",
					Description: "playlist description",
					Required:    true,
				},
				/*
					{
						Type:        discordgo.ApplicationCommandOptionBoolean,
						Name:        "private?",
						Description: "activate playlist privacy",
						Required:    true,
					},
				*/
			},
		},
		{
			Name:        "remove_playlist",
			Description: "Command for removing a playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "id",
					Description: "playlist string",
					Required:    true,
				},
			},
		},
		{
			Name:        "show_playlists",
			Description: "Command for adding a playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user",
					Description: "query an user's playlists",
					Required:    false,
				},
			},
		},
		{
			Name:        "search",
			Description: "search a string in all playlists",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "string",
					Description: "query an user's playlists",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user",
					Description: "query an user's playlists",
					Required:    false,
				},
			},
		},
		{
			Name:        "search_in_playlist",
			Description: "search a string in one specific playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "string",
					Description: "query an user's playlists",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "playlist-id",
					Description: "query an user's playlists",
					Required:    true,
				},
			},
		},
		{
			Name:        "get_random",
			Description: "get random video from all videos on registered playlists",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "string",
					Description: "query an user's playlists",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user",
					Description: "query an user's playlists",
					Required:    false,
				},
			},
		},
		{
			Name:        "random_from_playlist",
			Description: "get a random video from a specific playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "string",
					Description: "query an user's playlists",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "playlist-id",
					Description: "query an user's playlists",
					Required:    true,
				},
			},
		},
		{
			Name:        "refresh_playlist",
			Description: "refresh a playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "playlist-id",
					Description: "query an user's playlists",
					Required:    true,
				},
			},
		},

		/*
			{
				Name:        "auto_add_everything",
				Description: "Command for adding a playlist",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "channel id",
						Description: "use channel id to add everything",
						Required:    true,
					},
				},
			},
			{
				Name:        "set-playlist-privacy",
				Description: "Command for adding a playlist",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "playlist-id",
						Description: "use channel id to add everything",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "playlist-id",
						Description: "use channel id to add everything",
						Required:    true,
					},
				},
			},
		*/
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot){ //TODO check if playlist is empty
		"add_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			//check if playlist is on db
			//check and add user
			//check and add playlist
			//add videos
			//link video with playlist on the junction table
			ctx := context.Background()
			now := time.Now()
			options := i.ApplicationCommandData().Options
			command := "add_playlist"

			err := b.DB.NewSelect().Model(&Playlist{}).Where("id = ?", options[0].StringValue()).Scan(ctx)
			if err == nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist already added.", int(discordgo.InteractionResponseChannelMessageWithSource)))
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("fetching videos from playlist..", int(discordgo.InteractionResponseChannelMessageWithSource)))

			tx, err := b.DB.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while initializing a transaction", command)
				return
			}

			err = b.DB.NewSelect().Model(&User{}).Where("id = ?", i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_, err := tx.NewInsert().Model(&User{ID: i.Member.User.ID, Name: i.Member.Nick, Updated_at: &now, Created_at: &now}).Exec(ctx)
					if err != nil {
						_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
						log.Err(err).Msgf("[%s] error while adding new user on tx", command)
						return
					}
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
					log.Err(err).Msgf("[%s] error while checking if user exists.", command)
					return
				}
			}

			videos, err := fetchVideos(options[0].StringValue())
			if err != nil {
				log.Err(err).Msg("[%s] error while fetching videos from youtube") //
				if strings.HasPrefix(err.Error(), "The playlist identified") {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource)))
					return
				}
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				return
			}

			tx.NewInsert().Model(&Playlist{ID: options[0].StringValue(), Title: options[1].StringValue(), Description: options[2].StringValue(), Thumbnail: (*videos)[0].Thumbnail, Updated_at: &now, Created_at: &now}).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while inserting playlist on tx", command)
				return
			}

			tx.NewInsert().Model(videos).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while inserting videos on tx", command)
				return
			}

			var junctionTable []PlaylistVideo
			for _, v := range *videos {
				junctionTable = append(junctionTable, PlaylistVideo{video: v.ID, playlist: options[0].StringValue()})
			}

			tx.NewInsert().Model(&junctionTable).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while inserting playlist_video junction table to tx", command)
				return
			}

			err = tx.Commit()
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while committing tx", command)
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist added successfully.", int(discordgo.InteractionResponseChannelMessageWithSource)))
		},
		"remove_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options

			var playlists []Playlist
			err := b.DB.NewSelect().Model(&playlists).Where("user = ? AND id = ?", i.Member.User.ID, options[0].StringValue()).Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("user has no playlists registered.", int(discordgo.InteractionResponseChannelMessageWithSource)))
				return
			}

			_, err = b.DB.NewDelete().Model(&playlists).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[remove_playlist] error while deleting playlistis")
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("Playlist was deleted succesfully", int(discordgo.InteractionResponseChannelMessageWithSource)))

		},
		"show_playlists": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options
			user := i.Member.User.ID
			if options[0].StringValue() != "" {
				user = options[0].StringValue()
			}

			var playlists []Playlist
			err := b.DB.NewSelect().Model(&playlists).Where("user = ?", user).Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("user has no playlists registered.", int(discordgo.InteractionResponseChannelMessageWithSource)))
				return
			}

			//

		},
		"refresh_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			//check if user has playlist
			//get refreshed videos
			//get current videos linked with playlist
			//check new and deleted videos
			//update junction table
			//run a goroutine to delete videos with no playlists
			command := "refresh_playlist"
			ctx := context.Background()
			options := i.ApplicationCommandData().Options

			var playlist Playlist
			err := b.DB.NewSelect().Model(&playlist).Where("id = ? AND user = ?", options[0].StringValue(), i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource)))
					return
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
					log.Err(err).Msgf("[%s] error while checking if playlist exists.", command)
					return
				}
			}

			if (*playlist.Refreshed_at).Add(time.Hour * 24).After(time.Now()) {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("you can refresh a playlist only once a day.", int(discordgo.InteractionResponseChannelMessageWithSource)))
				return
			}

			videos, err := fetchVideos(options[0].StringValue())
			if err != nil {
				log.Err(err).Msgf("[%s] error while fetching videos from youtube", command) //
				if strings.HasPrefix(err.Error(), "The playlist identified") {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource)))
					return
				}
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				return
			}

			var junctionTable []PlaylistVideo
			err = b.DB.NewSelect().Model(&junctionTable).Where("id = ?", options[0].StringValue(), i.Member.User.ID).Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while requesting junction table entries", command)
				return
			}

			videosSET := make(map[string]struct{})
			for _, j := range junctionTable {
				videosSET[j.video] = struct{}{}
			}

			var videosToADD []Video
			var junctionTableToADD []PlaylistVideo
			for _, v := range *videos {
				if _, ok := videosSET[v.ID]; ok {
					delete(videosSET, v.ID) //remove the ones that are already on the table so i can delete the rest
				} else {
					videosToADD = append(videosToADD, v)
					junctionTableToADD = append(junctionTableToADD, PlaylistVideo{video: v.ID, playlist: options[0].StringValue()})
				}
			}

			tx, err := b.DB.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while initializing a transaction on refresh_playlist command.", command)
				return
			}

			_, err = tx.NewInsert().Model(&videosToADD).On("CONFLICT (id) DO UPDATE").Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while inserting videos on tx", command)
				return
			}

			_, err = tx.NewInsert().Model(&junctionTableToADD).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while inserting junction table on tx", command)
				return
			}

			var junctionTableToRemove []PlaylistVideo
			for k := range videosSET {
				junctionTable = append(junctionTable, PlaylistVideo{video: k, playlist: options[0].StringValue()})
			}

			_, err = tx.NewDelete().Model(&junctionTableToRemove).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while removing junction table on tx", command)
				return
			}

			_, err = tx.NewUpdate().Model((*Playlist)(nil)).Where("id=?", options[0].StringValue()).Set("last_refresh=?", time.Now()).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while inserting junction table on tx", command)
				return
			}

			err = tx.Commit()
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msgf("[%s] error while committing tx", command)
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist refreshed successfully.", int(discordgo.InteractionResponseChannelMessageWithSource)))

			go func() {

			}()

		},
	}
)

func (b *Bot) interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_, ok := i.Interaction.Data.(discordgo.MessageComponentInteractionData)
	if !ok {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i, b)
		}
		return
	}
}

func (b *Bot) newEmbededInteraction(title, footer, content string, mC []discordgo.MessageComponent, respType int) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseType(respType),
		Data: &discordgo.InteractionResponseData{
			Title: title,
			Flags: 64,
			Embeds: []*discordgo.MessageEmbed{&discordgo.MessageEmbed{
				Description: content,
				Color:       4989874,
				Footer:      &discordgo.MessageEmbedFooter{Text: footer},
			}},
			Components: messageComponents{discordgo.ActionsRow{
				Components: mC,
			},
			},
		},
	}
}

func (b *Bot) newSimpleInteraction(content string, respType int) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseType(respType),
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   64,
		},
	}
}

func (b *Bot) newButton(label, customID, url string, style discordgo.ButtonStyle) discordgo.Button {
	return discordgo.Button{
		Label:    label,
		Style:    style,
		CustomID: customID,
		URL:      url,
	}
}
