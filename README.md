## Installation

```
go get github.com/MixinNetwork/bot-api-go-client
```

## Quick Start

```golang
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
)

const (
	appId         = ""
	appSessionId  = ""
	appPrivateKey = ``
)

func main() {
	ctx := context.Background()
	// Generate Ed25519 key pair.
	pub, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", err
	}
	sessionSecret := base64.RawURLEncoding.EncodeToString(pub[:])
	// Rigster the user on the Mixin network
	user, err := bot.CreateUser(ctx, sessionSecret, "fullname", appId, appSessionId, appPrivateKey)
	if err != nil {
		fmt.Println(err)
		return
	}
	userSessionKey := base64.RawURLEncoding.EncodeToString(privateKey)
	// encrypt PIN
	encryptedPIN, err := bot.EncryptEd25519PIN(pin, user.PINTokenBase64, userSessionKey, uint64(time.Now().UnixNano()))
	if err != nil {
		return err
	}
	fmt.Println(encryptedPIN)
	// Set initial code.
	err = bot.UpdatePin(ctx, "", encryptedPIN, user.UserId, user.SessionId, userSessionKey)
	if err != nil {
		fmt.Println(err)
		return
	}
	//Sign authentication token.
	authenticationToken, err := bot.SignAuthenticationToken(user.UserId, user.SessionId, userSessionKey, "GET", "/assets", "")
	if err != nil {
		fmt.Println(err)
		return
	}
	// Read asset list
	assets, err := bot.AssetList(ctx, authenticationToken)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, a := range assets {
		fmt.Println(a.AssetId)
	}
}

```

Fo more examples, see [examples](https://github.com/MixinNetwork/bot-api-go-client/blob/master/examples/wallet.go)。
