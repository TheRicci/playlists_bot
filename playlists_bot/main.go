package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	youtube "google.golang.org/api/youtube/v3"
)

type Bot struct {
	*discordgo.Session
	*database
	randomMap map[string]*[]videoQuery //no need to use mutex, every key will only be accessed by one user from one command
}

func newBot() Bot {
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatal().Msgf("error while instantiating Discord bot : %s ", err.Error())
	}

	return Bot{
		dg,
		newDB(),
		make(map[string]*[]videoQuery),
	}
}

func playlistVideos(service *youtube.Service, part, playlistId, token string) (*youtube.PlaylistItemListResponse, error) {
	call := service.PlaylistItems.List([]string{part}).PlaylistId(playlistId).MaxResults(50)

	if token != "" {
		call = call.PageToken(token)
	}

	return call.Do()
}

func fetchVideos(playlistID string) (*[]Video, error) {
	ctx := context.Background()
	youtubeService, err := youtube.NewService(ctx, option.WithAPIKey(os.Getenv("YOUTUBE_KEY")))
	if err != nil {
		log.Fatal().Msgf("Error creating YouTube client: %v", err)
	}

	var videos []Video
	pageToken := ""
	now := time.Now()
	m := make(map[string]struct{})
	for {
		response, err := playlistVideos(youtubeService, "snippet", playlistID, pageToken)
		if err != nil {
			return nil, err
		}

		for _, item := range response.Items {
			if item.Snippet.Title == "Private video" || item.Snippet.Title == "Deleted video" {
				continue
			}
			if _, ok := m[item.Snippet.ResourceId.VideoId]; ok {
				continue
			} else {
				m[item.Snippet.ResourceId.VideoId] = struct{}{}
			}

			fmt.Println(item.Snippet.ResourceId.VideoId, ": ", item.Snippet.Title)
			videos = append(videos, Video{
				ID:            item.Snippet.ResourceId.VideoId,
				Title:         item.Snippet.Title,
				Thumbnail:     item.Snippet.Thumbnails.Medium.Url,
				Channel_id:    item.Snippet.ChannelId,
				Channel_title: item.Snippet.ChannelTitle,
				Updated_at:    &now,
				Created_at:    &now,
			})
		}

		if response.NextPageToken == "" {
			break
		}

		pageToken = response.NextPageToken
	}

	return &videos, nil
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal().Msgf("Error loading .env file")
	}

	bot := newBot()
	err = bot.Open()
	if err != nil {
		log.Fatal().Msgf("error while opening connection with Discord : %s ", err.Error())
	}

	log.Info().Msg("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := bot.ApplicationCommandCreate(bot.State.User.ID, "", v)
		if err != nil {
			log.Panic().Msgf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	bot.AddHandler(bot.interactionHandler)

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	log.Info().Msg("Removing commands...")
	for _, v := range registeredCommands {
		err := bot.ApplicationCommandDelete(bot.State.User.ID, "", v.ID)
		if err != nil {
			log.Fatal().Err(err).Msgf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}

	_ = bot.Close()
	log.Info().Msg("Bot exiting")
}
