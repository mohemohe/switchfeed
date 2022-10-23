package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/huandu/facebook/v2"
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
			Message     string `json:"message"`
			Application struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				NameSpace string `json:"namespace"`
				Category  string `json:"category"`
				Link      string `json:"link"`
			} `json:"application"`
			Attachments *struct {
				Data []struct {
					SubAttachments struct {
						Data []struct {
							Target []struct {
								ID string `json:"id"`
							} `json:"target"`
						} `json:"data"`
					} `json:"subattachments"`
				} `json:"data"`
			} `json:"attachments,omitempty"`
		} `json:"data"`
		Paging struct {
			Previous string `json:"previous"`
			Next     string `json:"next"`
		} `json:"paging"`
	}
	Image struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Images []struct {
			Width  int    `json:"width"`
			Height int    `json:"height"`
			Source string `json:"source"`
		} `json:"images"`
	}
	Result struct {
		Message string
		Images  []Image
	}
)

var (
	sess            *facebook.Session
	_latestObjectID uint64
	mu              sync.Mutex
)

func getLatestObjectID() uint64 {
	mu.Lock()
	defer mu.Unlock()

	id := _latestObjectID
	return id
}

func setLatestObjectID(id uint64) {
	mu.Lock()
	defer mu.Unlock()

	_latestObjectID = id
}

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

	result, err := getImageURLs()
	if err != nil {
		return
	}

	filePaths := make([]string, len(result.Images))
	for i, image := range result.Images {
		filePath, err := saveImage(image.ID, image.Images[0].Source)
		if err != nil {
			return
		}
		filePaths[i] = *filePath
	}

	if env.Mode.Mastodon {
		postMastodon(env, result.Message, filePaths)
	}
	if !env.Mode.Save {
		for _, filePath := range filePaths {
			deleteFile(filePath)
		}
	}
}

func getImageURLs() (*Result, error) {
	feedResult, err := sess.Get("/me/feed", M{
		"fields": "application,object_id,message,attachments",
	})
	if err != nil {
		log.Println("feed fetch error:", err)
		return nil, err
	}

	feed := new(Feed)
	if err := feedResult.Decode(feed); err != nil {
		log.Println("feed decode error:", err)
		return nil, err
	}
	if len(feed.Data) == 0 {
		return nil, err
	}
	latestObjectID := ""
	message := ""
	targetObjectIDs := make([]string, 0)
	for _, v := range feed.Data {
		if v.Application.NameSpace == "nintendoswitchshare" {
			latestObjectID = v.ObjectID
			message = v.Message
			if v.Attachments == nil {
				targetObjectIDs = append(targetObjectIDs, v.ObjectID)
			} else {
				for _, a := range v.Attachments.Data {
					for _, d := range a.SubAttachments.Data {
						for _, t := range d.Target {
							targetObjectIDs = append(targetObjectIDs, t.ID)
						}
					}
				}
			}
			break
		}
	}
	if latestObjectID == "" {
		return nil, err
	}

	latestUint64ObjectID, err := strconv.ParseUint(latestObjectID, 10, 64)
	if latestUint64ObjectID <= getLatestObjectID() {
		return nil, errors.New("already posted")
	}
	setLatestObjectID(latestUint64ObjectID)

	images := make([]Image, len(targetObjectIDs))
	for i, id := range targetObjectIDs {
		imageResult, err := sess.Get(id, M{
			"fields": "name,images",
		})
		if err != nil {
			log.Println("image list fetch error:", err)
			return nil, err
		}

		image := new(Image)
		if err := imageResult.Decode(image); err != nil {
			log.Println("feed decode error:", err)
			return nil, err
		}
		if len(image.Images) == 0 {
			return nil, err
		}
		sort.Slice(image.Images, func(i, j int) bool {
			return image.Images[i].Width > image.Images[j].Width
		})
		if message == "" && image.Name != "" {
			// FIXME: なんかこれでもtext取れたり取れなかったりするんだよなぁ
			message = image.Name
		}
		images[i] = *image
	}

	return &Result{
		Message: message,
		Images:  images,
	}, nil
}
