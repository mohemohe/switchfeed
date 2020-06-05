package main

import (
	"encoding/json"
	"github.com/huandu/facebook/v2"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"
)

type (
	WebHook struct {
		Object string `json:"object"`
		Entry []struct {
			ID string `json:"id"`
			UID string `json:"uid"`
			Time json.Number `json:"time"`
			Changes M `json:"changes"`
		} `json:"entry"`
	}
	Feed struct {
		Data []struct {
			ID string `json:"id"`
			ObjectID string `json:"object_id"`
			Application struct {
				ID string `json:"id"`
				Name string `json:"name"`
				NameSpace string `json:"namespace"`
				Category string `json:"category"`
				Link string `json:"link"`
			} `json:"application"`
		} `json:"data"`
		Paging struct {
			Previous string `json:"previous"`
			Next string `json:"next"`
		} `json:"paging"`
	}
	Image struct {
		ID string `json:"id"`
		Images []struct{
			Width int `json:"width"`
			Height int `json:"height"`
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
		"grant_type": "fb_exchange_token",
		"client_id": env.AppID,
		"client_secret": env.AppSecret,
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
		if diff.Hours() < 24 * 7 {
			cred = mustRefreshToken(env, cred.Token)
			saveCredential(cred)
			initFacebookSession(env, cred)
		}
		time.Sleep(time.Minute)
	}
}

func handleImage() {
	feedResult, err := sess.Get("/me/feed", M{
		"fields": "application,object_id",
	})
	if err != nil {
		log.Println("feed fetch error:", err)
		return
	}

	feed := new(Feed)
	if err := feedResult.Decode(feed); err != nil {
		log.Println("feed decode error:", err)
		return
	}
	if len(feed.Data) == 0 {
		return
	}
	latestObjectID := ""
	for _, v := range feed.Data {
		if v.Application.NameSpace == "nintendoswitchshare" {
			latestObjectID = v.ObjectID
			break
		}
	}
	if latestObjectID == "" {
		return
	}

	imageResult, err := sess.Get(latestObjectID, M{
		"fields": "images",
	})
	if err != nil {
		log.Println("image list fetch error:", err)
		return
	}

	image := new(Image)
	if err := imageResult.Decode(image); err != nil {
		log.Println("feed decode error:", err)
		return
	}
	if len(image.Images) == 0 {
		return
	}
	sort.Slice(image.Images, func(i, j int) bool {
		return image.Images[i].Width > image.Images[j].Width
	})
	imageURL := image.Images[0].Source

	resp, err := http.Get(imageURL)
	if err != nil {
		log.Println("image download error:", err)
		return
	}
	defer resp.Body.Close()

	u, err := url.Parse(imageURL)
	if err != nil {
		log.Println("image url parse error:", err)
		return
	}
	ext := filepath.Ext(u.Path)
	if ext == "" {
		ext = ".jpg" // NOTE: たぶん
	}

	filePath := path.Join(getDir(), "images", image.ID + ext)
	fi, _ := os.Stat(filePath)
	if fi != nil {
		return
	}

	file, err := os.Create(filePath)
	if err != nil {
		log.Println("file create error:", err)
		return
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		log.Println("image copy error:", err)
		return
	}
	log.Println("image downloaded:", path.Base(filePath))
}