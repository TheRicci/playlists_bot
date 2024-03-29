package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"google.golang.org/api/option"
	youtube "google.golang.org/api/youtube/v3"
)

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
				Channel_id:    item.Snippet.VideoOwnerChannelId,
				Channel_title: item.Snippet.VideoOwnerChannelTitle,
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
