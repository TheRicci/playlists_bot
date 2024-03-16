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

type User struct {
	ID         int64
	Name       string
	Updated_at *time.Time
	Created_at *time.Time
}

type Playlist struct {
	ID          string
	Title       string
	link        bool
	Is_private  bool
	Description string
	Updated_at  *time.Time
	Created_at  *time.Time
}

type Video struct {
	ID          string
	Title       string
	link        bool
	Description string
	Updated_at  *time.Time
	Created_at  *time.Time
}

var (
	integerOptionMinValue = float64(721)
	commands              = []*discordgo.ApplicationCommand{
		{
			Name:        "add-playlist",
			Description: "Command for adding a playlist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "playist",
					Description: "playlist string",
					Required:    true,
				},
			},
		},
		{
			Name:        "show-playlists",
			Description: "",
		},
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot){
		"add-playlist": func(s *discordgo.Session, i *discordgo.InteractionCreate, b *Bot) {
			ctx := context.Background()
			options := i.ApplicationCommandData().Options

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "fetching videos from playlist..",
					Flags:   64,
				},
			})

			tx, err := b.DB.BeginTx(context.Background(), &sql.TxOptions{})

			b.DB.NewSelect().Model(&User{}).Where("id = ?", i.Member.User.ID).Scan(ctx)

			if err != nil {
				if err == sql.ErrNoRows {
					tx.NewInsert().Model(&User{}).Exec(ctx)
				}
				log.Err(err).Msg("error while checking if user exists.")
				return
			}

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

func (b *Bot) newInteraction(title, footer, content string, mC []discordgo.MessageComponent, respType int) *discordgo.InteractionResponse {
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

func (b *Bot) newButton(label, customID, url string, style discordgo.ButtonStyle) discordgo.Button {
	return discordgo.Button{
		Label:    label,
		Style:    style,
		CustomID: customID,
		URL:      url,
	}
}
