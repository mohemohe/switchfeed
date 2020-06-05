package main

import (
	"context"
	"github.com/mattn/go-mastodon"
	"log"
)

func postMastodon(env *Env, text string, filePath string) {
	client := mastodon.NewClient(&mastodon.Config{
		Server: env.Mastodon.BaseURL,
		AccessToken: env.Mastodon.AccessToken,
	})

	attachment, err := client.UploadMedia(context.TODO(), filePath)
	if err != nil {
		log.Println("mastodon file upload error:", filePath, err)
		return
	}
	_, err = client.PostStatus(context.TODO(), &mastodon.Toot{
		Status:      text,
		MediaIDs: []mastodon.ID{
			attachment.ID,
		},

		// TODO: いつか対応する
		// NOTE: 共有するほど承認欲求が強いならpublicでええやろがい
		// Visibility:  "unlisted",
	})
	if err != nil {
		log.Println("mastodon post status error:", filePath, err)
	}
}