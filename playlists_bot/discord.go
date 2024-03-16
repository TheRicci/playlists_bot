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
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "private?",
					Description: "activate playlist privacy",
					Required:    true,
				},
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
			ctx := context.Background()
			now := time.Now()
			options := i.ApplicationCommandData().Options

			err := b.DB.NewSelect().Model(&Playlist{}).Where("id = ?", options[0].StringValue()).Scan(ctx)
			if err == nil {
				log.Err(err).Msg("error while reacting to define-start-channel command")
			}

			_ = s.InteractionRespond(i.Interaction, b.newSimpleInteraction("fetching videos from playlist..", int(discordgo.InteractionResponseChannelMessageWithSource)))

			tx, err := b.DB.BeginTx(context.Background(), &sql.TxOptions{})

			if err != nil {
				log.Err(err).Msg("error while reacting to define-start-channel command")
			}

			err = b.DB.NewSelect().Model(&User{}).Where("id = ?", i.Member.User.ID).Scan(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					tx.NewInsert().Model(&User{ID: i.Member.User.ID, Name: i.Member.Nick, Updated_at: &now, Created_at: &now}).Exec(ctx)
				}
				log.Err(err).Msg("error while checking if user exists.")
				return
			}

			videos := fetchVideos(options[0].StringValue())

			b.DB.NewInsert().Model()

			if err != nil {
				log.Err(err).Msg("error while reacting to define-start-channel command")
			}

			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "playlist added successfully.",
					Flags:   64,
				},
			})
			if err != nil {
				log.Err(err).Msg("error while reacting to define-start-channel command")
			}

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
