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

func playlistVideos(service *youtube.Service, part, playlistId, token string) *youtube.PlaylistItemListResponse {
	call := service.PlaylistItems.List([]string{part}).PlaylistId(playlistId).MaxResults(50)

	if token != "" {
		call = call.PageToken(token)
	}

	response, err := call.Do()
	handleError(err, "")
	return response
}

func fetchVideos(playlistID string) []Video {
	ctx := context.Background()
	youtubeService, err := youtube.NewService(ctx, option.WithAPIKey(os.Getenv("YOUTUBE_KEY")))
	if err != nil {
		log.Fatalf("Error creating YouTube client: %v", err)
	}

	var videos []Video
	pageToken := ""
	now := time.Now()
	for true {
		response := playlistVideos(youtubeService, "snippet", playlistID, pageToken)

		for _, item := range response.Items {
			fmt.Println(item.Id, ": ", item.Snippet.Title)
			videos = append(videos, Video{
				ID:          item.Snippet.Title,
				Title:       item.Snippet.Title,
				Description: item.Snippet.Description,
				Updated_at:  &now,
				Created_at:  &now,
			})
		}

		if response.NextPageToken == "" {
			break
		}

		pageToken = response.NextPageToken
	}

	return videos
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

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}
