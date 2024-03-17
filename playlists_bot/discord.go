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
	integerOptionMinValue = float64(721)
	commands              = []*discordgo.ApplicationCommand{
		{
			Name:        "add-playlist",
			Description: "Command for adding a playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "ID",
					Description: "playlist string",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "Title",
					Description: "playlist title",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "Description",
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
			Name:        "show-playlists",
			Description: "display playlists",
		},
		{
			Name:        "auto-add-everything",
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
					Name:        "playlist id",
					Description: "use channel id to add everything",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "playlist id",
					Description: "use channel id to add everything",
					Required:    true,
				},
			},
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot){
		"add-playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			//check if playlist is on db
			//check and add user
			//check and add playlist
			//add videos
			//link video with playlist on the junction table
			ctx := context.Background()
			now := time.Now()
			options := i.ApplicationCommandData().Options

			err := b.DB.NewSelect().Model(&Playlist{}).Where("id = ?", options[0].StringValue()).Scan(ctx)
			if err == nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist already added.", int(discordgo.InteractionResponseChannelMessageWithSource)))
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("fetching videos from playlist..", int(discordgo.InteractionResponseChannelMessageWithSource)))

			tx, err := b.DB.BeginTx(context.Background(), &sql.TxOptions{})
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msg("error while initializing a transaction on add-playlist command.")
				return
			}

			err = b.DB.NewSelect().Model(&User{}).Where("id = ?", i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					_, err := tx.NewInsert().Model(&User{ID: i.Member.User.ID, Name: i.Member.Nick, Updated_at: &now, Created_at: &now}).Exec(ctx)
					if err != nil {
						_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
						log.Err(err).Msg("error while adding new user to tx")
						return
					}
				} else {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
					log.Err(err).Msg("error while checking if user exists.")
					return
				}
			}

			videos, err := fetchVideos(options[0].StringValue())
			if err != nil {
				log.Err(err).Msg("error while initializing a transaction on add-playlist command.") //
				if strings.HasPrefix(err.Error(), "The playlist identified") {
					_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist not found.", int(discordgo.InteractionResponseChannelMessageWithSource)))
					return
				}
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				return
			}

			tx.NewInsert().Model(&Playlist{ID: options[0].StringValue(), Title: options[1].StringValue(), Description: options[2].StringValue(), Updated_at: &now, Created_at: &now}).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msg("error while inserting playlist to tx")
				return
			}

			tx.NewInsert().Model(videos).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msg("error while inserting videos to tx")
				return
			}

			var junctionTable []PlaylistVideo
			for _, v := range *videos {
				junctionTable = append(junctionTable, PlaylistVideo{video: v.ID, playlist: options[0].StringValue()})
			}

			tx.NewInsert().Model(&junctionTable).Exec(ctx)
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msg("error while inserting playlist_video junction table to tx")
				return
			}

			err = tx.Commit()
			if err != nil {
				_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("internal error", int(discordgo.InteractionResponseChannelMessageWithSource)))
				log.Err(err).Msg("error while committing tx")
				return
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("playlist added successfully.", int(discordgo.InteractionResponseChannelMessageWithSource)))
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
	switch strings.Split(mC.CustomID, "_")[0] {
	case "start":

	case "new-address":

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
