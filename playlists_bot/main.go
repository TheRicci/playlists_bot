package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/api/option"
	youtube "google.golang.org/api/youtube/v3"
)

func playlistsList(service *youtube.Service, part string, playlistId string) *youtube.PlaylistItemListResponse {
	call := service.PlaylistItems.List([]string{part}).PlaylistId(playlistId).MaxResults(50)

	response, err := call.Do()
	handleError(err, "")
	return response
}

func main() {
	err := godotenv.Load(".env")
	ctx := context.Background()
	youtubeService, err := youtube.NewService(ctx, option.WithAPIKey(os.Getenv("YOUTUBE_KEY")))
	if err != nil {
		log.Fatalf("Error creating YouTube client: %v", err)
	}

	response := playlistsList(youtubeService, "snippet", "PLi0LVkw6nHlvQlZ56C-LoKsRV_v0wpbPb")

	for _, item := range response.Items {

		fmt.Println(item.Id, ": ", item.Snippet.Title)
	}
}

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}
