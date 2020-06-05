package main

import (
	"os"
	"path"
)

type (
	Env struct {
		AppID string
		AppSecret string
		BaseURL string
	}
)

func mustEnv() *Env {
	appID := os.Getenv("FACEBOOK_APP_ID")
	appSecret := os.Getenv("FACEBOOK_APP_SECRET")
	baseUrl := os.Getenv("SWITCHFEED_BASE_URL")
	if appID == "" || appSecret == "" || baseUrl == "" {
		panic("環境変数 'FACEBOOK_APP_ID', 'FACEBOOK_APP_SECRET', 'SWITCHFEED_BASE_URL' が設定されていません")
	}

	return &Env{
		AppID:     appID,
		AppSecret: appSecret,
		BaseURL:   baseUrl,
	}
}

func getDir() string {
	return path.Dir(os.Args[0])
}

