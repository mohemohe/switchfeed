package main

import (
	"encoding/json"
	"fmt"
	"github.com/huandu/facebook/v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type (
	M = map[string]interface{}

	Env struct {
		AppID string
		AppSecret string
		BaseURL string
	}
	Credential struct {
		ExpireAt time.Time
		Token string
	}
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

	app := facebook.New(env.AppID, env.AppSecret)
	sess = app.Session(cred.Token)

	result, err := sess.Post(sess.App().AppId + "/subscriptions", M{
		"object": "user",
		"callback_url": env.BaseURL + "/webhook",
		"include_values": true,
		"fields": []string{
			"feed",
			"photos",
			"videos",
		},
		"access_token": app.AppAccessToken(),
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

func listen() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})
	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "code: " + r.URL.Query().Get("code"))
	})
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("post body read error:", err)
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)
		w.Write([]byte(r.URL.Query().Get("hub.challenge")))

		feed := new(WebHook)
		_ = json.Unmarshal(bytes, feed)
		if feed.Object == "" {
			return
		}

		if r.Method == http.MethodPost {
			go saveImage()
		}
	})
	log.Println("server listen on :8080")
	http.ListenAndServe(":8080", nil)
}

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

func getDir() string {
	return path.Dir(os.Args[0])
}

func getCredJsonPath() string {
	return path.Join(getDir(), "config", "credential.json")
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
		}
		time.Sleep(time.Minute)
	}
}

func saveImage() {
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