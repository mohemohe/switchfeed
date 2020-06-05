package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type (
	M = map[string]interface{}
)

func listen() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})
	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("code: " + r.URL.Query().Get("code")))
	})
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("post body read error:", err)
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)
		_, _ = w.Write([]byte(r.URL.Query().Get("hub.challenge")))

		feed := new(WebHook)
		_ = json.Unmarshal(bytes, feed)
		if feed.Object == "" {
			return
		}

		if r.Method == http.MethodPost {
			go handleImage()
		}
	})
	log.Println("server listen on :8080")
	log.Fatalln(http.ListenAndServe(":8080", nil))
}
