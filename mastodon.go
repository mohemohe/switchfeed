package main

import (
	"context"
	"log"

	"github.com/mattn/go-mastodon"
)

func postMastodon(env *Env, text string, filePaths []string) {
	client := mastodon.NewClient(&mastodon.Config{
		Server:      env.Mastodon.BaseURL,
		AccessToken: env.Mastodon.AccessToken,
	})

	mediaIDs := make([]mastodon.ID, len(filePaths))
	for i, filePath := range filePaths {
		attachment, err := client.UploadMedia(context.TODO(), filePath)
		if err != nil {
			log.Println("mastodon file upload error:", filePath, err)
			return
		}
		log.Println("mastodon file uploaded:", attachment.ID)
		mediaIDs[i] = attachment.ID
	}

	status, err := client.PostStatus(context.TODO(), &mastodon.Toot{
		Status:   text,
		MediaIDs: mediaIDs,

		// TODO: いつか対応する
		// NOTE: 共有するほど承認欲求が強いならpublicでええやろがい
		// Visibility:  "unlisted",
	})
	if err != nil {
		log.Println("mastodon post status error:", err)
		return
	}
	log.Println("mastodon posted:", status.ID, status.Content)
}
