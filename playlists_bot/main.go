package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	youtube "google.golang.org/api/youtube/v3"
)

type Bot struct {
	*discordgo.Session
	*database
}

func newBot() Bot {
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatalf("error while instantiating Discord bot : %s ", err.Error())
	}

	return Bot{
		dg,
		newDB(),
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
		log.Fatalf("Error creating YouTube client: %v", err)
	}

	var videos []Video
	pageToken := ""
	now := time.Now()
	for true {
		response, err := playlistVideos(youtubeService, "snippet", playlistID, pageToken)

		if err != nil {
			return nil, err
		}

		for _, item := range response.Items {
			fmt.Println(item.Id, ": ", item.Snippet.Title)
			videos = append(videos, Video{
				ID:          item.Snippet.PlaylistId,
				Title:       item.Snippet.Title,
				Description: item.Snippet.Description,
				Thumbnail:   item.Snippet.Thumbnails.High.Url,
				Updated_at:  &now,
				Created_at:  &now,
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
		log.Fatalf("Error loading .env file")
	}
	//fetchVideos()

	bot := newBot()
	err = bot.Open()
	if err != nil {
		log.Fatalf("error while opening connection with Discord : %s ", err.Error())
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	_ = bot.Close()

}
