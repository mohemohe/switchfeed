package main

import (
	"encoding/json"
	"fmt"
	"github.com/huandu/facebook/v2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"
)

type (
	Credential struct {
		ExpireAt time.Time
		Token string
	}
)

func newCredential(env *Env) *Credential {
	app := facebook.New(env.AppID, env.AppSecret)
	app.RedirectUri = env.BaseURL + "/token"

	time.Sleep(2 * time.Second) // HACK: server listen on〜に邪魔されないように

	fmt.Println("https://www.facebook.com/dialog/oauth?client_id=" + env.AppID + "&redirect_uri=" + app.RedirectUri + "&scope=public_profile,user_posts,user_photos,user_videos")
	fmt.Println("にアクセスしてログインコードを取得してください")
	fmt.Printf("code: ")
	var code string
	_, _ = fmt.Scan(&code)

	token, err := app.ParseCode(code)
	if err != nil {
		log.Println("ログインコードのパースに失敗しました:", code, err)
		os.Exit(2)
	}

	return mustRefreshToken(env, token)
}

func loadCredential() *Credential {
	bytes, err := ioutil.ReadFile(getCredJsonPath())
	if err != nil {
		return nil
	}
	cred := new(Credential)
	if err := json.Unmarshal(bytes, cred); err != nil {
		return nil
	}
	return cred
}

func saveCredential(cred *Credential) {
	bytes, err := json.Marshal(cred)
	if err != nil {
		return
	}
	if err := ioutil.WriteFile(getCredJsonPath(), bytes, 0600); err != nil {
		log.Println("credential write error:", err)
	}
}

func getCredJsonPath() string {
	return path.Join(getDir(), "config", "credential.json")
}