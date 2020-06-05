package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type (
	Env struct {
		AppID     string
		AppSecret string
		BaseURL   string
		Mode      Mode
		Mastodon  Mastodon
	}
	Mode struct {
		Save     bool
		Mastodon bool
	}
	Mastodon struct {
		BaseURL     string
		AccessToken string
	}
)

func mustEnv() *Env {
	appID := os.Getenv("FACEBOOK_APP_ID")
	appSecret := os.Getenv("FACEBOOK_APP_SECRET")
	baseURL := os.Getenv("SWITCHFEED_BASE_URL")
	mode := os.Getenv("SWITCHFEED_MODE")
	mastodonBaseURL := os.Getenv("MASTODON_BASE_URL")
	mastodonAccessToken := os.Getenv("MASTODON_ACCESS_TOKEN")
	if appID == "" || appSecret == "" || baseURL == "" || mode == "" {
		panic("環境変数 'FACEBOOK_APP_ID', 'FACEBOOK_APP_SECRET', 'SWITCHFEED_BASE_URL', 'SWITCHFEED_MODE' のどれか、あるいは全てが設定されていません")
	}
	modes := strings.Split(mode, ",")
	m := Mode{
		Save:     isSaveMode(modes),
		Mastodon: isMastodonMode(modes),
	}
	if m.Mastodon {
		if mastodonBaseURL == "" || mastodonAccessToken == "" {
			panic("環境変数 'MASTODON_BASE_URL', 'MASTODON_ACCESS_TOKEN' のどれか、あるいは全てが設定されていません")
		}
	}
	return &Env{
		AppID:     appID,
		AppSecret: appSecret,
		BaseURL:   baseURL,
		Mode:      m,
		Mastodon: Mastodon{
			BaseURL:     mastodonBaseURL,
			AccessToken: mastodonAccessToken,
		},
	}
}

func getDir() string {
	return path.Dir(os.Args[0])
}

func isSaveMode(modes []string) bool {
	for _, mode := range modes {
		if strings.TrimSpace(mode) == "save" {
			return true
		}
	}
	return false
}

func isMastodonMode(modes []string) bool {
	for _, mode := range modes {
		if strings.TrimSpace(mode) == "mastodon" {
			return true
		}
	}
	return false
}

func saveImage(id string, imageURL string) (*string, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		log.Println("image download error:", err)
		return nil, err
	}
	defer resp.Body.Close()

	u, err := url.Parse(imageURL)
	if err != nil {
		log.Println("image url parse error:", err)
		return nil, err
	}
	ext := filepath.Ext(u.Path)
	if ext == "" {
		ext = ".jpg" // NOTE: たぶん
	}

	filePath := path.Join(getDir(), "images", id+ext)
	fi, _ := os.Stat(filePath)
	if fi != nil {
		return nil, err
	}

	file, err := os.Create(filePath)
	if err != nil {
		log.Println("file create error:", err)
		return nil, err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		log.Println("image copy error:", err)
		return nil, err
	}
	log.Println("image downloaded:", path.Base(filePath))

	return &filePath, nil
}

func deleteFile(filePath string) {
	if err := os.Remove(filePath); err != nil {
		log.Println("file delete error:", filePath, err)
	}
}
