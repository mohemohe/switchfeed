# switchfeed

![](https://i.imgur.com/04TIfKr.png)

FacebookにシェアしたSwitchの画像を自動でダウンロードします  
動画はまだ対応していません

## 動かし方

```bash
git clone https://github.com/mohemohe/switchfeed.git
cd switchfeed
cp sample.env .env
# 好みのエディタで.envを書き換える
docker-compose run --service-ports app
# OAuth認証をしてcodeを入れる
# success:true が表示されたらCtrl+Cで落とす
docker-compose up -d
```

## Facebookアプリケーションの設定方法

- `webhookに使うドメイン名` を `.env` に入力する
- https://developers.facebook.com/apps/ でアプリを作成する
- `アプリID` と `app secret` を `.env` に入力する
- `アプリ左メニュー > 設定 > ベーシック` で `アプリドメイン` に `webhookに使うドメイン名` を入力する

![](https://i.imgur.com/wcjzc7Z.png)

- `アプリ左メニュー > Facebookログイン > 設定` で `有効なOAuthリダイレクトURI` に `https://【webhookに使うドメイン名】/token` を入力する

![](https://i.imgur.com/hhNsp2J.png)

## こんなときは

### 画像が取得できない

`user_posts` `user_photos` `user_videos` の権限は、未申請のFacebookアプリだと開発者しか使えないぞい  
Switchに紐づけたFacebookアカウントを使うとええ

### 認証をやりなおしたい

```bash
rm -f config/credential.json
```

### アクセストークンが自動更新されなかった

ロングタームトークンは60日くらいあるはずだけど、60日も動かしてテストしてないから、ぶっちゃけトークン更新が動くか分からん