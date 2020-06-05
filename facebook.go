package main

import (
	"encoding/json"
	"github.com/huandu/facebook/v2"
	"log"
	"os"
	"sort"
	"time"
)

type (
	WebHook struct {
		Object string `json:"object"`
		Entry  []struct {
			ID      string      `json:"id"`
			UID     string      `json:"uid"`
			Time    json.Number `json:"time"`
			Changes M           `json:"changes"`
		} `json:"entry"`
	}
	Feed struct {
		Data []struct {
			ID          string `json:"id"`
			ObjectID    string `json:"object_id"`
			Application struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				NameSpace string `json:"namespace"`
				Category  string `json:"category"`
				Link      string `json:"link"`
			} `json:"application"`
		} `json:"data"`
		Paging struct {
			Previous string `json:"previous"`
			Next     string `json:"next"`
		} `json:"paging"`
	}
	Image struct {
		ID     string `json:"id"`
		Images []struct {
			Width  int    `json:"width"`
			Height int    `json:"height"`
			Source string `json:"source"`
		} `json:"images"`
	}
)

var sess *facebook.Session

func initFacebookSession(env *Env, cred *Credential) {
	app := facebook.New(env.AppID, env.AppSecret)
	sess = app.Session(cred.Token)
}

func mustRefreshToken(env *Env, token string) *Credential {
	app := facebook.New(env.AppID, env.AppSecret)
	tmpSess := app.Session(token)
	res, err := tmpSess.Get("/oauth/access_token", M{
		"grant_type":        "fb_exchange_token",
		"client_id":         env.AppID,
		"client_secret":     env.AppSecret,
		"fb_exchange_token": token,
	})
	if err != nil {
		log.Println("ロングターム トークンの取得に失敗しました:", err)
		os.Exit(3)
	}
	expiresIn, _ := res.Get("expires_in").(json.Number).Int64()
	expireAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	cred := &Credential{
		ExpireAt: expireAt,
		Token:    res.Get("access_token").(string),
	}
	saveCredential(cred)
	return cred
}

func watchToken(env *Env, cred *Credential) {
	for {
		diff := cred.ExpireAt.Sub(time.Now())
		if diff.Hours() < 24*7 {
			cred = mustRefreshToken(env, cred.Token)
			saveCredential(cred)
			initFacebookSession(env, cred)
		}
		time.Sleep(time.Minute)
	}
}

func handleImage() {
	env := mustEnv()
	shouldHandle := env.Mode.Save || env.Mode.Mastodon
	if !shouldHandle {
		return
	}

	id, url, err := getImageURL()
	if err != nil {
		return
	}

	filePath, err := saveImage(*id, *url)
	if err != nil {
		return
	}
	if env.Mode.Mastodon {
		postMastodon(env, "", *filePath)
	}
	if !env.Mode.Save {
		deleteFile(*filePath)
	}
}

func getImageURL() (*string, *string, error) {
	feedResult, err := sess.Get("/me/feed", M{
		"fields": "application,object_id",
	})
	if err != nil {
		log.Println("feed fetch error:", err)
		return nil, nil, err
	}

	feed := new(Feed)
	if err := feedResult.Decode(feed); err != nil {
		log.Println("feed decode error:", err)
		return nil, nil, err
	}
	if len(feed.Data) == 0 {
		return nil, nil, err
	}
	latestObjectID := ""
	for _, v := range feed.Data {
		if v.Application.NameSpace == "nintendoswitchshare" {
			latestObjectID = v.ObjectID
			break
		}
	}
	if latestObjectID == "" {
		return nil, nil, err
	}

	imageResult, err := sess.Get(latestObjectID, M{
		"fields": "images",
	})
	if err != nil {
		log.Println("image list fetch error:", err)
		return nil, nil, err
	}

	image := new(Image)
	if err := imageResult.Decode(image); err != nil {
		log.Println("feed decode error:", err)
		return nil, nil, err
	}
	if len(image.Images) == 0 {
		return nil, nil, err
	}
	sort.Slice(image.Images, func(i, j int) bool {
		return image.Images[i].Width > image.Images[j].Width
	})

	return &image.ID, &image.Images[0].Source, nil
}
