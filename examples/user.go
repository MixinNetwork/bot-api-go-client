package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client/v3"
)

func main() {
	// flags: -keystore path, -name for new user
	ksPath := flag.String("keystore", "", "path to keystore JSON downloaded from developers.mixin.one")
	name := flag.String("name", "example-user", "full name for the new user")
	spend := flag.String("spend", "", "optional spend private key (hex) to register safe with pin setup")
	flag.Parse()

	if *ksPath == "" {
		log.Fatalf("please provide -keystore path")
	}

	dat, err := os.ReadFile(*ksPath)
	if err != nil {
		log.Fatalf("read keystore: %v", err)
	}
	var su bot.SafeUser
	if err := json.Unmarshal(dat, &su); err != nil {
		log.Fatalf("parse keystore: %v", err)
	}

	// generate a temporary session key pair (use the public as session_secret)
	seed := make([]byte, ed25519.SeedSize)
	// for example purposes, use timestamp-derived seed (NOT for production)
	copy(seed, []byte(fmt.Sprintf("seed-%d", time.Now().UnixNano())))
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	sessionPublic := base64.RawURLEncoding.EncodeToString(pub)

	ctx := context.Background()

	// Call CreateUserSimple to create a bare user
	user, err := bot.CreateUserSimple(ctx, sessionPublic, *name)
	if err != nil {
		log.Fatalf("CreateUserSimple error: %v", err)
	}
	fmt.Printf("created user: %+v\n", user)

	// If we have a spend key, attempt to register safe with setup pin
	if *spend != "" {
		safeUser := &bot.SafeUser{
			UserId:            user.UserId,
			SessionId:         user.SessionId,
			SessionPrivateKey: hex.EncodeToString(seed),
			ServerPublicKey:   user.ServerPublicKey,
			SpendPrivateKey:   *spend,
		}

		regUser, err := bot.RegisterSafeWithSetupPin(ctx, safeUser)
		if err != nil {
			log.Fatalf("RegisterSafeWithSetupPin error: %v", err)
		}
		fmt.Printf("registered safe user: %+v\n", regUser)
	} else {
		fmt.Println("no -spend provided, skipping RegisterSafeWithSetupPin")
	}
}
