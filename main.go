package main

import (
	"log"
	"os"
	"path"
	"sync"
)

func main() {
	_ = os.Mkdir(path.Join(getDir(), "config"), 0700)
	_ = os.Mkdir(path.Join(getDir(), "images"), 0755)

	go listen()

	env := mustEnv()
	cred := loadCredential()
	if cred != nil {
		mustRefreshToken(env, cred.Token)
	} else {
		cred = newCredential(env)
	}
	go watchToken(env, cred)

	initFacebookSession(env, cred)

	result, err := sess.Post(sess.App().AppId+"/subscriptions", M{
		"object":         "user",
		"callback_url":   env.BaseURL + "/webhook",
		"include_values": true,
		"fields": []string{
			"feed",
			"photos",
			"videos",
		},
		"access_token": sess.App().AppAccessToken(),
		"verify_token": "switchfeed", // FIXME
	})
	if err != nil {
		log.Println("subscribe error:", err)
	}
	log.Println(result)

	w := new(sync.WaitGroup)
	w.Add(1)
	w.Wait()
}
