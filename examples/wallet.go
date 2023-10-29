package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/MixinNetwork/bot-api-go-client/v2"
)

const (
	uid        = ""
	sid        = ""
	sessionKey = ``
)

func main() {
	ctx := context.Background()

	user, userSessionKey, err := createUser(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(user.UserId)
	fmt.Println(userSessionKey)
	err = setupPin(ctx, "123456", user, userSessionKey)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Setup PIN successful")
	ethAssetId := "4d8c508b-91c5-375b-92b0-ee702ed2dac5"
	asset, err := getAsset(ctx, user, userSessionKey, ethAssetId)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(asset)
	assets, err := getAssets(ctx, user, userSessionKey)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, a := range assets {
		fmt.Println(a.AssetId)
	}
}

func createUser(ctx context.Context) (*bot.User, string, error) {
	pub, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", err
	}
	sessionSecret := base64.RawURLEncoding.EncodeToString(pub[:])
	user, err := bot.CreateUser(ctx, sessionSecret, "TestName", uid, sid, sessionKey)
	if err != nil {
		return nil, "", err
	}
	userSessionKey := base64.RawURLEncoding.EncodeToString(privateKey)
	return user, userSessionKey, nil
}

func setupPin(ctx context.Context, pin string, user *bot.User, userSessionKey string) error {
	encryptedPIN, err := bot.EncryptEd25519PIN(pin, user.PINTokenBase64, userSessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return err
	}
	err = bot.UpdatePin(ctx, "", encryptedPIN, user.UserId, user.SessionId, userSessionKey)
	if err != nil {
		return err
	}
	return nil
}

func getAsset(ctx context.Context, user *bot.User, userSessionKey, assetId string) (*bot.Asset, error) {
	token, err := bot.SignAuthenticationToken(user.UserId, user.SessionId, userSessionKey, "GET", "/assets/"+assetId, "")
	if err != nil {
		return nil, err
	}
	fmt.Println(token)
	return bot.AssetShow(ctx, assetId, token)
}

func getAssets(ctx context.Context, user *bot.User, userSessionKey string) ([]*bot.Asset, error) {
	token, err := bot.SignAuthenticationToken(user.UserId, user.SessionId, userSessionKey, "GET", "/assets", "")
	if err != nil {
		return nil, err
	}
	fmt.Println(token)
	return bot.AssetList(ctx, token)
}
