package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

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
					Name:        "playlist-id",
					Description: "query an user's playlists",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "string",
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
			command, errorString := "add_playlist", "Internal error."

			err := b.DB.NewSelect().Model(&Playlist{}).Where("id = ?", options[0].StringValue()).Scan(ctx)
			if err == nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist already added.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("fetching videos from playlist..", int(discordgo.InteractionResponseChannelMessageWithSource), 64))

			tx, err := b.DB.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while initializing a transaction", command)
				return
			}

			err = b.DB.NewSelect().Model(&User{}).Where("id = ?", i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_, err := tx.NewInsert().Model(&User{ID: i.Member.User.ID, Name: i.Member.User.Username, Updated_at: &now, Created_at: &now}).Exec(ctx)
					if err != nil {
						_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
						log.Err(err).Msgf("[%s] error while adding new user on tx", command)
						return
					}
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					log.Err(err).Msgf("[%s] error while checking if user exists.", command)
					return
				}
			}

			videos, err := fetchVideos(options[0].StringValue())
			if err != nil {
				log.Err(err).Msg("[%s] error while fetching videos from youtube") //
				if strings.HasPrefix(err.Error(), "The playlist identified") {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				}
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				return
			}

			_, err = tx.NewInsert().Model(&Playlist{ID: options[0].StringValue(), User_id: i.Member.User.ID, Title: options[1].StringValue(), Description: options[2].StringValue(), Thumbnail: (*videos)[0].Thumbnail, Updated_at: &now, Created_at: &now}).Exec(ctx)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while inserting playlist on tx", command)
				return
			}

			_, err = tx.NewInsert().Model(videos).Exec(ctx)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while inserting videos on tx", command)
				return
			}

			var junctionTable []PlaylistVideo
			for _, v := range *videos {
				junctionTable = append(junctionTable, PlaylistVideo{Video_id: v.ID, Playlist_id: options[0].StringValue()})
			}

			_, err = tx.NewInsert().Model(&junctionTable).Exec(ctx)
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while inserting playlist_video junction table to tx", command)
				return
			}

			err = tx.Commit()
			if err != nil {
				_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &errorString})
				log.Err(err).Msgf("[%s] error while committing tx", command)
				return
			}

			ok := "playlist added successfully."
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &ok})
		},
		"remove_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options

			var playlists []Playlist
			err := b.DB.NewSelect().Model(&playlists).Where("user_id = ? AND id = ?", i.Member.User.ID, options[0].StringValue()).Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not registered.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			_, err = b.DB.NewDelete().Model(&playlists).WherePK().Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[remove_playlist] error while deleting playlistis")
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("Playlist was deleted succesfully", int(discordgo.InteractionResponseChannelMessageWithSource), 64))

		},
		"show_playlists": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options
			user := i.Member.User
			if len(options) != 0 {
				user, _ = s.User(options[0].StringValue())
			}

			var playlists []Playlist
			err := b.DB.NewSelect().Model(&playlists).Where("user_id = ?", user.ID).Scan(ctx)
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
					Flags: discordgo.MessageFlags(8),
					Embeds: append(embed, &discordgo.MessageEmbed{
						Title:     fmt.Sprintf("%s's Playlists", strings.Title(strings.ToLower(user.Username))),
						Thumbnail: &discordgo.MessageEmbedThumbnail{URL: user.AvatarURL("")},
						Fields:    fields,
					}),
				},
			})

		},
		"search_in_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options
			command := "search_in_playlist"

			var playlist Playlist
			err := b.DB.NewSelect().Model(&playlist).Where("id = ? AND user_id = ?", options[0].StringValue(), i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					log.Err(err).Msgf("[%s] error while checking if playlist exists.", command)
					return
				}
			}

			var videos []videoQuery
			err = b.DB.NewSelect().Model(&videos).
				ColumnExpr("v.id").
				ColumnExpr("v.title").
				ColumnExpr("v.description").
				Join("JOIN \"playlistsDB_video\" AS v ON v.id = video_query.video_id").
				Where("video_query.playlist_id = ? AND LOWER(v.title) SIMILAR TO ?", options[0].StringValue(), "%( |^)"+strings.ToLower(options[1].StringValue())+"( |$)%").
				Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while checking if playlist exists.", command)
				return
			}

			if len(videos) == 0 {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("no videos found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
			}

			emj := discordgo.ComponentEmoji{ID: "1221585609907372104", Name: "vhss"}

			// separate query result in different lists of options
			var list []*[]discordgo.SelectMenuOption
			var menuOptions []discordgo.SelectMenuOption
			j := 0
			for i2, v := range videos {
				menuOptions = append(menuOptions, discordgo.SelectMenuOption{
					Label:       v.Title,
					Value:       fmt.Sprint(i2),
					Emoji:       emj,
					Description: fmt.Sprintf("from channel: %s", v.Channel_title),
				})

				if i2 == len(videos)-1 {
					list = append(list, &menuOptions)
					b.openCommandSearch[i.Member.User.ID] = MenuSelectionState{
						maxIndex:     j,
						currentIndex: 0,
						videos:       videos,
						list:         list,
					}
				} else if i2%24 == 0 && i2 > 0 {
					list = append(list, &menuOptions)
					menuOptions = make([]discordgo.SelectMenuOption, 0)
					j++
				}
			}
			var componentsList []discordgo.MessageComponent
			var button discordgo.Button
			componentsList = append(componentsList, b.newSelectMenu("search_select_menu", (*list[0])[1:]))
			if len(list) > 1 {
				button = b.newButton("", "next_search_list", discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "button", ID: "1222350837406371880"})
				componentsList = append(componentsList, button)
			}

			state := b.openCommandSearch[i.Member.User.ID]
			state.currentButtons = []discordgo.Button{button}
			b.openCommandSearch[i.Member.User.ID] = state

			components := &messageComponents{discordgo.ActionsRow{
				Components: componentsList,
			}}

			s.InteractionRespond(i.Interaction, b.newInteraction("search", int(discordgo.InteractionResponseChannelMessageWithSource), b.newEmbed(
				videos[0].Title,
				videos[0].Channel_title,
				videos[0].ID,
				videos[0].Thumbnail), *components),
			)

		},
		"refresh_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) { //TODO check if playlist is empty
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
			err := b.DB.NewSelect().Model(&playlist).Where("id = ? AND user_id = ?", options[0].StringValue(), i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					log.Err(err).Msgf("[%s] error while checking if playlist exists.", command)
					return
				}
			}

			if (*playlist.Last_refresh).Add(time.Hour * 24).After(time.Now()) {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("you can refresh a playlist only once a day.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				return
			}

			videos, err := fetchVideos(options[0].StringValue())
			if err != nil {
				log.Err(err).Msgf("[%s] error while fetching videos from youtube", command) //
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
				log.Err(err).Msgf("[%s] error while requesting junction table entries", command)
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
				log.Err(err).Msgf("[%s] error while initializing a transaction on refresh_playlist command.", command)
				return
			}

			_, err = tx.NewInsert().Model(&videosToADD).On("CONFLICT (id) DO UPDATE").Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while inserting videos on tx", command)
				return
			}

			_, err = tx.NewInsert().Model(&junctionTableToADD).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while inserting junction table on tx", command)
				return
			}

			var junctionTableToRemove []PlaylistVideo
			for k := range videosSET {
				junctionTable = append(junctionTable, PlaylistVideo{Video_id: k, Playlist_id: options[0].StringValue()})
			}

			_, err = tx.NewDelete().Model(&junctionTableToRemove).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while removing junction table on tx", command)
				return
			}

			_, err = tx.NewUpdate().Model((*Playlist)(nil)).Where("id=?", options[0].StringValue()).Set("last_refresh=?", time.Now()).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while inserting junction table on tx", command)
				return
			}

			err = tx.Commit()
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while committing tx", command)
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist refreshed successfully.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))

			go func() {
				//deleting videos with no playlists
				_, err := b.DB.NewDelete().NewRaw("DELETE FROM video WHERE id NOT IN (SELECT DISTINCT video_id FROM playlist_video);").Exec(ctx)
				if err != nil {
					log.Err(err).Msgf("[%s] error while deleting dangling videos", command)
				}
			}()

		},
		"random_from_playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options
			command := "random_from_playlist"

			var Playlist Playlist
			err := b.DB.NewSelect().Model(&Playlist).Where("id = ? AND user_id = ?", options[0].StringValue(), i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					return
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
					log.Err(err).Msgf("[%s] error while checking if playlist exists.", command)
					return
				}
			}

			var videoQuery []videoQuery
			err = b.DB.NewSelect().Model(&videoQuery).
				ColumnExpr("v.id").
				ColumnExpr("v.title").
				ColumnExpr("v.description").
				ColumnExpr("v.thumbnail").
				ColumnExpr("v.channel_title").
				Join("JOIN \"playlistsDB_video\" AS v ON v.id = video_query.video_id").
				Where("video_query.playlist_id = ?", options[0].StringValue()).
				OrderExpr("RANDOM()").
				Scan(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource), 64))
				log.Err(err).Msgf("[%s] error while checking if playlist exists.", command)
				return
			}

			list := videoQuery[1:]
			b.randomMap[fmt.Sprintf("%s-new_random_from_playlist", i.Member.User.ID)] = &list

			components := messageComponents{discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					b.newButton("Next Random", "new_random_from_playlist", discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "bluestar", ID: "1221587912861417613"})},
			}}

			err = s.InteractionRespond(i.Interaction, b.newInteraction("random", int(discordgo.InteractionResponseChannelMessageWithSource), b.newEmbed(
				videoQuery[0].Title,
				videoQuery[0].Channel_title,
				videoQuery[0].ID,
				videoQuery[0].Thumbnail), components),
			)
			fmt.Println(err)
		},
	}
)

func (b *Bot) interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	mC, ok := i.Interaction.Data.(discordgo.MessageComponentInteractionData)
	if !ok {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i, b)
		}
		return
	}

	fmt.Println(mC.CustomID)

	switch mC.CustomID {
	case "new_random_from_playlist":
		lenVideoArray := len(*b.randomMap[fmt.Sprintf("%s-%s", i.Member.User.ID, mC.CustomID)])
		if lenVideoArray == 0 {
			return
		}

		video := (*b.randomMap[fmt.Sprintf("%s-%s", i.Member.User.ID, mC.CustomID)])[0]
		var components messageComponents

		if lenVideoArray == 1 {
			components = messageComponents{}
		} else {
			remaining := (*b.randomMap[fmt.Sprintf("%s-%s", i.Member.User.ID, mC.CustomID)])[1:]
			b.randomMap[fmt.Sprintf("%s-%s", i.Member.User.ID, mC.CustomID)] = &remaining

			components = messageComponents{discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					b.newButton("Next Random",
						"new_random_from_playlist",
						discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "bluestar", ID: "1221587912861417613"})},
			}}
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "wait", CustomID: "wait"},
		})

		embed := b.newEmbed(
			video.Title,
			video.Channel_title,
			video.ID,
			video.Thumbnail)
		s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Components: components,
			Embeds:     []*discordgo.MessageEmbed{&embed},
			ID:         i.Message.ID,
			Channel:    i.ChannelID,
		})
	case "search_select_menu":
		fmt.Println(mC.Values)
		menuState := b.openCommandSearch[i.Member.User.ID]
		videoIndex, _ := strconv.Atoi(mC.Values[0])
		list := *menuState.list[menuState.currentIndex]
		actualVideoIndex := videoIndex + (menuState.currentIndex * 24)

		comps := []discordgo.MessageComponent{b.newSelectMenu("search_select_menu", append(list[:actualVideoIndex], list[actualVideoIndex+1:]...))}
		for _, b := range menuState.currentButtons {
			comps = append(comps, b)
		}
		components := &messageComponents{discordgo.ActionsRow{
			Components: comps,
		}}

		s.InteractionRespond(i.Interaction, b.newInteraction("search", int(discordgo.InteractionResponseChannelMessageWithSource), b.newEmbed(
			menuState.videos[actualVideoIndex].Title,
			menuState.videos[actualVideoIndex].Channel_title,
			menuState.videos[actualVideoIndex].ID,
			menuState.videos[actualVideoIndex].Thumbnail), *components),
		)

	case "next_search_list":
		menuState := b.openCommandSearch[i.Member.User.ID]
		menuState.currentIndex++
		b.openCommandSearch[i.Member.User.ID] = menuState
		b.searchMenu(i, s, menuState)

	case "previous_search_list":
		menuState := b.openCommandSearch[i.Member.User.ID]
		menuState.currentIndex--
		b.openCommandSearch[i.Member.User.ID] = menuState
		b.searchMenu(i, s, menuState)

	}

}

func (b *Bot) searchMenu(i *discordgo.InteractionCreate, s *discordgo.Session, menuState MenuSelectionState) {
	buttons := b.buttonsChange(menuState.currentIndex, menuState.maxIndex)
	list := *menuState.list[menuState.currentIndex]
	actualVideoIndex := menuState.currentIndex * 25

	comps := []discordgo.MessageComponent{b.newSelectMenu("search_select_menu", list[1:])}
	for _, b := range buttons {
		comps = append(comps, b)
	}
	components := &messageComponents{discordgo.ActionsRow{
		Components: comps,
	}}

	s.InteractionRespond(i.Interaction, b.newInteraction("search", int(discordgo.InteractionResponseChannelMessageWithSource), b.newEmbed(
		menuState.videos[actualVideoIndex].Title,
		menuState.videos[actualVideoIndex].Channel_title,
		menuState.videos[actualVideoIndex].ID,
		menuState.videos[actualVideoIndex].Thumbnail), *components),
	)
}

func (b *Bot) buttonsChange(currentIndex, maxIndex int) []discordgo.Button {
	buttons := []discordgo.Button{
		b.newButton("", "next_search_list", discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "button", ID: "1222350837406371880"}),
		b.newButton("", "previous_search_list", discordgo.PrimaryButton, discordgo.ComponentEmoji{Name: "button2", ID: "1222350851188854804"}),
	}
	if currentIndex == 0 {
		return buttons[:1]
	}
	if currentIndex == maxIndex {
		return buttons[1:]
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

func (b *Bot) newInteraction(title string, respType int, embed discordgo.MessageEmbed, mC []discordgo.MessageComponent) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseType(respType),
		Data: &discordgo.InteractionResponseData{
			Title:      title,
			Flags:      16,
			Embeds:     []*discordgo.MessageEmbed{&embed},
			Components: mC,
		},
	}
}

func (b *Bot) MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !m.Author.Bot {
		return
	}

	if len((*m.Message).Embeds) == 0 && (*m.Message).Type == 19 && (*m.Message).Content == "" {
		time.Sleep(time.Second)
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		return
	}

	if m.Interaction != nil && (*m.Interaction).Name == "random_from_playlist" {
		if m, ok := b.openCommandRandom[m.Interaction.User.ID]; ok {
			s.ChannelMessageDelete(m.ChannelID, m.ID)
		}

		b.openCommandRandom[m.Interaction.User.ID] = m.Message
	}

}
