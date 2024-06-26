package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
)

type messageComponents []discordgo.MessageComponent

type videoQuery struct {
	ID            string
	Title         string
	Description   string
	Thumbnail     string
	Channel_id    string
	Channel_title string
	bun.BaseModel `bun:"playlistsDB_playlist_video"`
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "add_playlist",
			Description: "command to add playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "link",
					Description: "playlist link",
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
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "is_private",
					Description: "activate playlist privacy",
					Required:    true,
				},
			},
		},
		{
			Name:        "remove_playlist",
			Description: "command to remove playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "playlist",
					Description: "title or id",
					Required:    true,
				},
			},
		},
		{
			Name:        "show_playlists",
			Description: "command to show playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user",
					Description: "specify a user",
					Required:    false,
				},
			},
		},
		{
			Name:        "show_private_playlists",
			Description: "command to show playlist",
		},
		{
			Name:        "search",
			Description: "search through all playlists",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "string",
					Description: "string to search through the playlists",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "include_private",
					Description: "true or false",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "query user's videos",
					Required:    false,
				},
			},
		},
		{
			Name:        "search_in_playlist",
			Description: "search through a specific playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "playlist",
					Description: "title or id",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "string",
					Description: "string to search through the playlist",
					Required:    true,
				},
			},
		},
		{
			Name:        "random",
			Description: "get a random video from all videos registered from user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "include_private",
					Description: "true or false",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "get random video",
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
					Name:        "playlist",
					Description: "title or id",
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
					Name:        "playlist",
					Description: "title or id",
					Required:    true,
				},
			},
		},

		/*
			{
				Name:        "auto_add_everything",
				Description: "i.ApplicationCommandData().Name for adding a playlist",
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
				Description: "i.ApplicationCommandData().Name for adding a playlist",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "playlist",
						Description: "title or id",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionBoolean,
						Name:        "private?",
						Description: "true or false",
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
			errorString := "Internal error."

			regex := regexp.MustCompile(`(?:https?:\/\/)?(?:www\.)?(?:youtube\.com\/(?:[^\/\n\s]+\/\S+\/|(?:v|e(?:mbed)?)\/|\S*?[?&]list=)|youtu\.be\/)([a-zA-Z0-9_-]{18,34})`)
			matches := regex.FindStringSubmatch(options[0].StringValue())

			if len(matches) < 2 {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("invalid link", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			err := b.DB.NewSelect().Model(&Playlist{}).Where("id = ?", matches[1]).Scan(ctx)
			if err == nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("title already exists", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			err = b.DB.NewSelect().Model(&Playlist{}).Where("title = ? AND user_id = ?", options[1].StringValue(), i.Member.User.ID).Scan(ctx)
			if err == nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist already added.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("fetching videos from playlist..", int(discordgo.InteractionResponseChannelMessageWithSource), 64))

			tx, err := b.DB.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while initializing a transaction", i.ApplicationCommandData().Name)
				return
			}

			err = b.DB.NewSelect().Model(&User{}).Where("id = ?", i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_, err := tx.NewInsert().Model(&User{ID: i.Member.User.ID, Name: i.Member.User.Username, Updated_at: &now, Created_at: &now}).Exec(ctx)
					if err != nil {
						_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
						log.Err(err).Msgf("[%s] error while adding new user on tx", i.ApplicationCommandData().Name)
						return
					}
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					log.Err(err).Msgf("[%s] error while checking if user exists.", i.ApplicationCommandData().Name)
					return
				}
			}

			videos, err := fetchVideos(matches[1])
			if err != nil {
				log.Err(err).Msg("[%s] error while fetching videos from youtube") //
				if strings.HasPrefix(err.Error(), "The playlist identified") {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				}
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				return
			}

			_, err = tx.NewInsert().Model(&Playlist{ID: matches[1], User_id: i.Member.User.ID, Title: options[1].StringValue(), Description: options[2].StringValue(), Thumbnail: (*videos)[0].Thumbnail, Is_private: options[3].BoolValue(), Updated_at: &now, Created_at: &now}).Exec(ctx)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while inserting playlist on tx", i.ApplicationCommandData().Name)
				return
			}

			_, err = tx.NewInsert().Model(videos).On("CONFLICT (id) DO UPDATE").Exec(ctx)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while inserting videos on tx", i.ApplicationCommandData().Name)
				return
			}

			var junctionTable []PlaylistVideo
			for _, v := range *videos {
				junctionTable = append(junctionTable, PlaylistVideo{Video_id: v.ID, Playlist_id: matches[1]})
			}

			_, err = tx.NewInsert().Model(&junctionTable).Exec(ctx)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while inserting playlist_video junction table to tx", i.ApplicationCommandData().Name)
				return
			}

			err = tx.Commit()
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while committing tx", i.ApplicationCommandData().Name)
				return
			}

			ok := "playlist added successfully."
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &ok})
		},
		"remove_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options

			var playlist Playlist
			err := b.DB.NewSelect().Model(&playlist).Where("user_id = ? AND (id = ? OR title = ?)", i.Member.User.ID, options[0].StringValue(), options[0].StringValue()).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not registered.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					log.Err(err).Msgf("[%s] error while checking if playlist exists.", i.ApplicationCommandData().Name)
					return
				}
			}

			tx, err := b.DB.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while initializing a transaction", i.ApplicationCommandData().Name)
				return
			}

			_, err = tx.NewDelete().Model((*PlaylistVideo)(nil)).Where("playlist_id = ?", options[0].StringValue()).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while deleting playlists in the juction table", i.ApplicationCommandData().Name)
				return
			}

			_, err = tx.NewDelete().Model(&playlist).WherePK().Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while deleting playlists", i.ApplicationCommandData().Name)
				return
			}

			err = tx.Commit()
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while commiting transaction", i.ApplicationCommandData().Name)
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("Playlist was deleted succesfully", int(discordgo.InteractionResponseChannelMessageWithSource), 64))

			go b.removeDanglingVideos()
		},
		"show_playlists": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			options := i.ApplicationCommandData().Options
			user := i.Member.User
			if len(options) != 0 {
				user, _ = s.User(options[0].StringValue())
			}

			b.showPlaylists(s, i, user, false, 8)

		},
		"show_private_playlists": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			b.showPlaylists(s, i, i.Member.User, true, 64)

		},
		"search": func(s *discordgo.Session, interaction *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := interaction.ApplicationCommandData().Options
			user := interaction.Member.User.ID
			private, user_set := false, false

			if len(options) > 1 {
				for _, o := range options {
					if o.Type == discordgo.ApplicationCommandOptionBoolean {
						private = o.BoolValue()
					} else {
						user = o.UserValue(s).ID
						user_set = true
					}
				}
			}
			if user_set && private {
				_ = s.InteractionRespond(interaction.Interaction, b.newSimpleInteraction("can't access private videos from another user", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
			}

			var videos []videoQuery
			query := b.DB.NewSelect().Model(&videos).
				ColumnExpr("v.id").
				ColumnExpr("v.title").
				ColumnExpr("v.thumbnail").
				ColumnExpr("v.channel_title").
				Join("JOIN \"playlistsDB_video\" AS v ON v.id = video_query.video_id").
				Join("JOIN \"playlistsDB_playlist\" AS p ON p.id = video_query.playlist_id").
				Where("p.user_id = ?", user)

			flags := 16
			if private {
				flags = 64
			} else {
				query = query.Where("p.is_private = false")
			}

			err := query.Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(interaction.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while checking if playlist exists.", interaction.ApplicationCommandData().Name)
				return
			}
			if len(videos) == 0 {
				_ = s.InteractionRespond(interaction.Interaction, b.newSimpleInteraction("no videos found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			components, video := b.searchFunction(options[0].StringValue(), interaction, videos)
			if components == nil {
				_ = s.InteractionRespond(interaction.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			s.InteractionRespond(interaction.Interaction, b.newInteraction("search", int(discordgo.InteractionResponseChannelMessageWithSource), b.newEmbed(
				video.Title,
				video.Channel_title,
				video.ID,
				video.Thumbnail), *components, flags),
			)

		},
		"search_in_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options

			var playlist Playlist
			err := b.DB.NewSelect().Model(&playlist).Where("(id = ? OR title = ?) AND user_id = ?", options[0].StringValue(), options[0].StringValue(), i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					log.Err(err).Msgf("[%s] error while checking if playlist exists.", i.ApplicationCommandData().Name)
					return
				}
			}

			flags := 16
			if playlist.Is_private {
				flags = 64
			}

			var videos []videoQuery
			err = b.DB.NewSelect().Model(&videos).
				ColumnExpr("v.id").
				ColumnExpr("v.title").
				ColumnExpr("v.thumbnail").
				ColumnExpr("v.channel_title").
				Join("JOIN \"playlistsDB_video\" AS v ON v.id = video_query.video_id").
				Join("JOIN \"playlistsDB_playlist\" AS p ON p.id = video_query.playlist_id").
				Where("p.id = ? OR p.title = ?", options[0].StringValue(), options[0].StringValue()).
				Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while checking if playlist exists.", i.ApplicationCommandData().Name)
				return
			}
			if len(videos) == 0 {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("no videos found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			components, video := b.searchFunction(options[1].StringValue(), i, videos)
			if components == nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			s.InteractionRespond(i.Interaction, b.newInteraction("search", int(discordgo.InteractionResponseChannelMessageWithSource), b.newEmbed(
				video.Title,
				video.Channel_title,
				video.ID,
				video.Thumbnail), *components, flags),
			)

		},
		"refresh_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) { //TODO check if playlist is empty
			//check if user has playlist
			//get refreshed videos
			//get current videos linked with playlist
			//check new and deleted videos
			//update junction table
			//run a goroutine to delete videos with no playlists
			ctx := context.Background()
			options := i.ApplicationCommandData().Options

			var playlist Playlist
			err := b.DB.NewSelect().Model(&playlist).Where("(id = ? OR title = ?) AND user_id = ?", options[0].StringValue(), options[0].StringValue(), i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					log.Err(err).Msgf("[%s] error while checking if playlist exists.", i.ApplicationCommandData().Name)
					return
				}
			}

			if (*playlist.Last_refresh).Add(time.Hour * 24).After(time.Now()) {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("you can refresh a playlist only once a day.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			videos, err := fetchVideos(options[0].StringValue())
			if err != nil {
				log.Err(err).Msgf("[%s] error while fetching videos from youtube", i.ApplicationCommandData().Name) //
				if strings.HasPrefix(err.Error(), "The playlist identified") {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				}
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			var junctionTable []PlaylistVideo
			err = b.DB.NewSelect().Model(&junctionTable).Where("id = ?", options[0].StringValue(), i.Member.User.ID).Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while requesting junction table entries", i.ApplicationCommandData().Name)
				return
			}

			videosSET := make(map[string]struct{})
			for _, j := range junctionTable {
				videosSET[j.Video_id] = struct{}{}
			}

			var videosToADD []Video
			var junctionTableToADD []PlaylistVideo
			for _, v := range *videos {
				if _, ok := videosSET[v.ID]; ok {
					delete(videosSET, v.ID) //remove the ones that are already on the table so i can delete the rest
				} else {
					videosToADD = append(videosToADD, v)
					junctionTableToADD = append(junctionTableToADD, PlaylistVideo{Video_id: v.ID, Playlist_id: options[0].StringValue()})
				}
			}

			tx, err := b.DB.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while initializing a transaction on refresh_playlist i.ApplicationCommandData().Name.", i.ApplicationCommandData().Name)
				return
			}

			_, err = tx.NewInsert().Model(&videosToADD).On("CONFLICT (id) DO UPDATE").Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while inserting videos on tx", i.ApplicationCommandData().Name)
				return
			}

			_, err = tx.NewInsert().Model(&junctionTableToADD).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while inserting junction table on tx", i.ApplicationCommandData().Name)
				return
			}

			var junctionTableToRemove []PlaylistVideo
			for k := range videosSET {
				junctionTable = append(junctionTable, PlaylistVideo{Video_id: k, Playlist_id: options[0].StringValue()})
			}

			_, err = tx.NewDelete().Model(&junctionTableToRemove).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while removing junction table on tx", i.ApplicationCommandData().Name)
				return
			}

			_, err = tx.NewUpdate().Model((*Playlist)(nil)).Where("id=?", options[0].StringValue()).Set("last_refresh=?", time.Now()).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while inserting junction table on tx", i.ApplicationCommandData().Name)
				return
			}

			err = tx.Commit()
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while committing tx", i.ApplicationCommandData().Name)
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist refreshed successfully.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))

			go b.removeDanglingVideos()

		},
		"random_from_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options

			var playlist Playlist
			err := b.DB.NewSelect().Model(&playlist).Where("(id = ? OR title = ?) AND user_id = ?", options[0].StringValue(), options[0].StringValue(), i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					log.Err(err).Msgf("[%s] error while checking if playlist exists.", i.ApplicationCommandData().Name)
					return
				}
			}

			flags := 16
			if playlist.Is_private {
				flags = 64
			}

			var videoQuery []videoQuery
			err = b.DB.NewSelect().Model(&videoQuery).
				ColumnExpr("v.id").
				ColumnExpr("v.title").
				ColumnExpr("v.thumbnail").
				ColumnExpr("v.channel_title").
				Join("JOIN \"playlistsDB_video\" AS v ON v.id = video_query.video_id").
				Join("JOIN \"playlistsDB_playlist\" AS p ON p.id = video_query.playlist_id").
				Where("p.id = ? OR p.title = ?", options[0].StringValue(), options[0].StringValue()).
				OrderExpr("RANDOM()").
				Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while checking if playlist exists.", i.ApplicationCommandData().Name)
				return
			}

			list := videoQuery[1:]
			b.randomMap[fmt.Sprintf("%s-new_random", i.Member.User.ID)] = &list

			components := messageComponents{discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					b.newButton("New Random", "new_random", discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "bluestar", ID: "1221587912861417613"})},
			}}

			s.InteractionRespond(i.Interaction, b.newInteraction("random", int(discordgo.InteractionResponseChannelMessageWithSource), b.newEmbed(
				videoQuery[0].Title,
				videoQuery[0].Channel_title,
				videoQuery[0].ID,
				videoQuery[0].Thumbnail), components, flags),
			)

		},
		"random": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options
			user := i.Member.User.ID
			private, user_set := false, false

			if len(options) != 0 {
				for _, o := range options {
					if o.Type == discordgo.ApplicationCommandOptionBoolean {
						private = o.BoolValue()
					} else {
						user = o.UserValue(s).ID
						user_set = true
					}
				}
			}
			if user_set && private {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("can't access private videos from another user", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
			}

			var videoQuery []videoQuery
			query := b.DB.NewSelect().Model(&videoQuery).
				ColumnExpr("v.id").
				ColumnExpr("v.title").
				ColumnExpr("v.thumbnail").
				ColumnExpr("v.channel_title").
				Join("JOIN \"playlistsDB_video\" AS v ON v.id = video_query.video_id").
				Join("JOIN \"playlistsDB_playlist\" AS p ON p.id = video_query.playlist_id").
				Where("p.user_id = ?", user)

			flags := 16
			if private {
				flags = 64
			} else {
				query = query.Where("p.private = false")
			}

			err := query.OrderExpr("RANDOM()").Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while checking if playlist exists.", i.ApplicationCommandData().Name)
				return
			}

			list := videoQuery[1:]
			b.randomMap[fmt.Sprintf("%s-new_random", i.Member.User.ID)] = &list

			components := messageComponents{discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					b.newButton("New Random", "new_random", discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "bluestar", ID: "1221587912861417613"})},
			}}

			s.InteractionRespond(i.Interaction, b.newInteraction("random", int(discordgo.InteractionResponseChannelMessageWithSource), b.newEmbed(
				videoQuery[0].Title,
				videoQuery[0].Channel_title,
				videoQuery[0].ID,
				videoQuery[0].Thumbnail), components, flags),
			)

		},
	}
)

func (b *Bot) showPlaylists(s *discordgo.Session, i *discordgo.InteractionCreate, user *discordgo.User, private bool, flags int) {
	ctx := context.Background()
	var playlists []Playlist
	err := b.DB.NewSelect().Model(&playlists).Where("user_id = ? AND is_private = ?", user.ID, private).Scan(ctx)
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("user has no playlists registered.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
		return
	}

	var fields []*discordgo.MessageEmbedField
	for i, p := range playlists {
		fields = append(fields, &discordgo.MessageEmbedField{Name: fmt.Sprintf("#%v %s", i+1, p.Title), Value: fmt.Sprintf("https://www.youtube.com/playlist?list=%s", p.ID)})
	}

	var embed []*discordgo.MessageEmbed
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlags(flags),
			Embeds: append(embed, &discordgo.MessageEmbed{
				Title:     fmt.Sprintf("%s's Playlists", strings.Title(strings.ToLower(user.Username))),
				Thumbnail: &discordgo.MessageEmbedThumbnail{URL: user.AvatarURL("")},
				Fields:    fields,
			}),
		},
	})

}

func (b *Bot) searchFunction(q string, interaction *discordgo.InteractionCreate, videos []videoQuery) (*messageComponents, *videoQuery) {
	indexMapping := bleve.NewIndexMapping()
	index, err := bleve.NewMemOnly(indexMapping)
	if err != nil {
		log.Err(err).Msgf("[%s] error while creating mem only index", interaction.ApplicationCommandData().Name)
		return nil, nil
	}

	for _, data := range videos {
		err := index.Index(data.Title, data)
		if err != nil {
			log.Err(err).Msgf("[%s] error while indexing.", interaction.ApplicationCommandData().Name)
			return nil, nil
		}
	}

	defer index.Close()

	query := bleve.NewQueryStringQuery(q)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = 200
	searchRequest.Fields = []string{"ID", "Title", "Thumbnail", "Channel_title"}
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		log.Err(err).Msgf("[%s] search error.", interaction.ApplicationCommandData().Name)
		return nil, nil
	}

	videosResult := make([]videoQuery, len(searchResult.Hits))

	for i, hit := range searchResult.Hits {
		videosResult[i].ID = hit.Fields["ID"].(string)
		videosResult[i].Title = hit.Fields["Title"].(string)
		videosResult[i].Thumbnail = hit.Fields["Thumbnail"].(string)
		videosResult[i].Channel_title = hit.Fields["Channel_title"].(string)
	}
	emj := discordgo.ComponentEmoji{ID: "1221585609907372104", Name: "vhss"}

	// separate query result in different lists of options
	var list []*[]discordgo.SelectMenuOption
	var menuOptions []discordgo.SelectMenuOption
	i := 0
	j := 0
	for i2, v := range videosResult {
		menuOptions = append(menuOptions, discordgo.SelectMenuOption{
			Label:       v.Title,
			Value:       fmt.Sprint(i),
			Emoji:       emj,
			Description: fmt.Sprintf("from channel: %s", v.Channel_title),
		})
		if i2 == len(videosResult)-1 {
			newOptions := menuOptions
			list = append(list, &newOptions)
			b.openCommandSearch[interaction.Member.User.ID] = MenuSelectionState{
				maxIndex:     j,
				currentIndex: 0,
				videos:       videosResult,
				list:         list,
			}
		} else if i == 24 {
			newOptions := menuOptions
			list = append(list, &newOptions)
			menuOptions = make([]discordgo.SelectMenuOption, 0)
			j++
			i = 0
			continue
		}
		i++
	}

	var button discordgo.Button
	var components messageComponents
	if len(videosResult) > 1 {
		components = messageComponents{discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{b.newSelectMenu("search_select_menu", (*list[0])[1:])},
		}}
		if len(list) > 1 {
			button = b.newButton("next", "next_search_list", discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "button", ID: "1222350837406371880"})
			components = append(components, discordgo.ActionsRow{Components: []discordgo.MessageComponent{button}})
		}
	}
	state := b.openCommandSearch[interaction.Member.User.ID]
	state.currentButtons = []discordgo.Button{button}
	b.openCommandSearch[interaction.Member.User.ID] = state

	return &components, &videosResult[0]
}

func (b *Bot) interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	mC, ok := i.Interaction.Data.(discordgo.MessageComponentInteractionData)
	if !ok {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i, b)
		}
		return
	}

	switch mC.CustomID {
	case "new_random":
		lenVideoArray := len(*b.randomMap[fmt.Sprintf("%s-%s", i.Member.User.ID, mC.CustomID)])
		if lenVideoArray == 0 {
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "wait1", CustomID: "wait"},
		})

		video := (*b.randomMap[fmt.Sprintf("%s-%s", i.Member.User.ID, mC.CustomID)])[0]
		var components messageComponents
		if lenVideoArray == 1 {
			components = messageComponents{}
		} else {
			remaining := (*b.randomMap[fmt.Sprintf("%s-%s", i.Member.User.ID, mC.CustomID)])[1:]
			b.randomMap[fmt.Sprintf("%s-%s", i.Member.User.ID, mC.CustomID)] = &remaining

			components = messageComponents{discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					b.newButton("New Random",
						"new_random",
						discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "bluestar", ID: "1221587912861417613"})},
			}}
		}

		embed := b.newEmbed(
			video.Title,
			video.Channel_title,
			video.ID,
			video.Thumbnail)
		s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Components: components,
			Embed:      &embed,
			ID:         i.Message.ID,
			Channel:    i.ChannelID,
		})
	case "search_select_menu":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "wait1", CustomID: "wait2", Title: "wait3"},
		})

		menuState := b.openCommandSearch[i.Member.User.ID]
		videoIndex, _ := strconv.Atoi(mC.Values[0])
		list := *menuState.list[menuState.currentIndex]
		actualVideoIndex := videoIndex + (menuState.currentIndex * 25)

		components := messageComponents{discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{b.newSelectMenu("search_select_menu", list)},
		}}
		var buttonsComps []discordgo.MessageComponent
		for _, b := range menuState.currentButtons {
			buttonsComps = append(buttonsComps, b)
		}

		components = append(components, discordgo.ActionsRow{Components: buttonsComps})
		embed := b.newEmbed(
			menuState.videos[actualVideoIndex].Title,
			menuState.videos[actualVideoIndex].Channel_title,
			menuState.videos[actualVideoIndex].ID,
			menuState.videos[actualVideoIndex].Thumbnail)
		s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Embed:      &embed,
			Components: components,
			Flags:      i.Message.Flags,
			ID:         i.Message.ID,
			Channel:    i.ChannelID,
		})
	case "next_search_list":
		menuState := b.openCommandSearch[i.Member.User.ID]
		menuState.currentIndex++
		menuState = b.searchMenu(i, s, menuState)
		b.openCommandSearch[i.Member.User.ID] = menuState

	case "previous_search_list":
		menuState := b.openCommandSearch[i.Member.User.ID]
		menuState.currentIndex--
		menuState = b.searchMenu(i, s, menuState)
		b.openCommandSearch[i.Member.User.ID] = menuState

	default:
		return
	}

	b.interactionEnd <- i.Message.ID
}

func (b *Bot) searchMenu(i *discordgo.InteractionCreate, s *discordgo.Session, menuState MenuSelectionState) MenuSelectionState {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "wait", CustomID: "wait"},
	})

	buttons := b.buttonsChange(menuState.currentIndex, menuState.maxIndex)
	menuState.currentButtons = buttons
	list := *menuState.list[menuState.currentIndex]
	actualVideoIndex := menuState.currentIndex * 25

	var components messageComponents
	if len(list) > 1 {
		components = messageComponents{discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{b.newSelectMenu("search_select_menu", list[1:])},
		}}
	}
	var buttonsComps []discordgo.MessageComponent
	for _, b := range buttons {
		buttonsComps = append(buttonsComps, b)
	}
	components = append(components, discordgo.ActionsRow{Components: buttonsComps})

	embed := b.newEmbed(
		menuState.videos[actualVideoIndex].Title,
		menuState.videos[actualVideoIndex].Channel_title,
		menuState.videos[actualVideoIndex].ID,
		menuState.videos[actualVideoIndex].Thumbnail)

	s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Embed:      &embed,
		Components: components,
		Flags:      i.Message.Flags,
		ID:         i.Message.ID,
		Channel:    i.ChannelID,
	})

	return menuState
}

func (b *Bot) buttonsChange(currentIndex, maxIndex int) []discordgo.Button {
	buttons := []discordgo.Button{
		b.newButton("previous", "previous_search_list", discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "button2", ID: "1222350851188854804"}),
		b.newButton("next", "next_search_list", discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "button", ID: "1222350837406371880"}),
	}
	if currentIndex == 0 {
		return buttons[1:]
	}
	if currentIndex == maxIndex {
		return buttons[:1]
	}

	return buttons
}

func (b *Bot) newEmbed(title, content, id, imageURL string) discordgo.MessageEmbed {
	return discordgo.MessageEmbed{
		URL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", id),
		Title:       title,
		Description: content,
		Image:       &discordgo.MessageEmbedImage{URL: imageURL, Height: 10, Width: 10},
	}
}

func (b *Bot) newSimpleInteraction(content string, respType int, flags int) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseType(respType),
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlags(flags),
		},
	}
}

func (b *Bot) newButton(label, customID string, style discordgo.ButtonStyle, emj discordgo.ComponentEmoji) discordgo.Button {
	return discordgo.Button{
		Emoji:    emj,
		Label:    label,
		Style:    style,
		CustomID: customID,
	}
}

func (b *Bot) newSelectMenu(customID string, options []discordgo.SelectMenuOption) discordgo.SelectMenu {
	return discordgo.SelectMenu{
		CustomID: customID,
		Options:  options,
	}

}

func (b *Bot) newInteraction(title string, respType int, embed discordgo.MessageEmbed, mC []discordgo.MessageComponent, flags int) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseType(respType),
		Data: &discordgo.InteractionResponseData{
			Title:      title,
			Flags:      discordgo.MessageFlags(flags),
			Embeds:     []*discordgo.MessageEmbed{&embed},
			Components: mC,
		},
	}
}

func (b *Bot) removeDanglingVideos() {
	//deleting videos with no playlists
	_, err := b.DB.NewDelete().NewRaw("DELETE FROM \"playlistsDB_video\" WHERE id NOT IN (SELECT DISTINCT video_id FROM \"playlistsDB_playlist_video\");").Exec(context.Background())
	if err != nil {
		log.Err(err).Msg("error while deleting dangling videos")
	}
}

func (b *Bot) MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !m.Author.Bot {
		return
	}

	if len((*m.Message).Embeds) == 0 && (*m.Message).Type == 19 && (*m.Message).Content == "" {
		for {
			select {
			case mID := <-b.interactionEnd:
				if mID != m.MessageReference.MessageID {
					continue
				}
				s.ChannelMessageDelete(m.ChannelID, m.ID)
				return
			}
		}

	}
}
