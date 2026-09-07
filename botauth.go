package bot

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/crypto/curve25519"
)

const (
	userPlatformPrefix = "up_"
)

type BotAuthClient struct {
	Cache    BotAuthCache
	SafeUser *SafeUser
	Logger   *slog.Logger
}

type BotAuthCache interface {
	Get(key string) ([]byte, error)
	Put(key string, value []byte) error
	Delete(key string) error
}

type MapCache struct {
	mu sync.RWMutex
	m  map[string][]byte
}

func NewMapCache() *MapCache {
	return &MapCache{
		m: map[string][]byte{},
	}
}

func (c *MapCache) Get(key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return bytes.Clone(c.m[key]), nil
}

func (c *MapCache) Put(key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = make(map[string][]byte)
	}
	c.m[key] = bytes.Clone(value)
	return nil
}

func (c *MapCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
	return nil
}

func NewBotAuthClient(cache BotAuthCache, su *SafeUser, logger *slog.Logger) *BotAuthClient {
	if cache == nil {
		cache = NewMapCache()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &BotAuthClient{
		Cache:    cache,
		SafeUser: su,
		Logger:   logger,
	}
}

func NewDefaultClient(su *SafeUser, logger *slog.Logger) *BotAuthClient {
	mapCache := NewMapCache()
	return NewBotAuthClient(mapCache, su, logger)
}

func (c *BotAuthClient) SignRequest(ctx context.Context, ts int64, botUserId string, r *http.Request) (string, error) {
	if c.SafeUser == nil {
		return "", fmt.Errorf("safe user is nil")
	}
	if r == nil {
		return "", fmt.Errorf("request is nil")
	}
	sharedKey, err := c.getSharedKey(ctx, botUserId)
	if err != nil {
		return "", errors.Errorf("failed to decode public key: %v", err)
	}
	if _, err := parseEd25519PrivateKey(c.SafeUser.SessionPrivateKey); err != nil {
		return "", err
	}

	data := fmt.Appendf(nil, "%d%s%s", ts, r.Method, r.URL.RequestURI())
	if r.Body != nil {
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r.Body)
		if err != nil {
			return "", errors.Errorf("failed to read body: %v", err)
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(buf.Bytes()))
		data = append(data, buf.Bytes()...)
	}
	hash, err := hex.DecodeString(HmacSha256(sharedKey, data))
	if err != nil {
		return "", errors.Errorf("failed to hash: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(fmt.Appendf(nil, "%s%s", c.SafeUser.UserId, hash)), nil
}

func (c *BotAuthClient) getSharedKey(ctx context.Context, userId string) ([]byte, error) {
	value, err := c.Cache.Get(userId)
	var sharedKey []byte
	if err != nil || value == nil || len(value) < 32 {
		c.Logger.Debug(fmt.Sprintf("cache miss for %s", userId))
		userSessions, err := FetchUserSessions(ctx, []string{userId}, c.SafeUser)
		if err != nil {
			return nil, err
		}
		var userSession *UserSession
		for _, us := range userSessions {
			if us != nil && (us.UserId == userId || us.UserId == "" && len(userSessions) == 1) {
				userSession = us
				break
			}
		}
		if userSession == nil {
			return nil, fmt.Errorf("userSession for %s nil", userId)
		}
		uPk, err := base64.RawURLEncoding.DecodeString(userSession.PublicKey)
		if err != nil {
			return nil, err
		}
		platform := userSession.Platform
		priv, err := parseEd25519PrivateKey(c.SafeUser.SessionPrivateKey)
		if err != nil {
			return nil, err
		}
		var p [32]byte
		PrivateKeyToCurve25519(&p, priv)
		sharedKey, err = curve25519.X25519(p[:], uPk[:])
		if err != nil {
			return nil, err
		}
		err = c.Cache.Put(userId, sharedKey[:])
		if err != nil {
			c.Logger.Warn(fmt.Sprintf("save shared key for %s error %v", userId, err))
		}
		err = c.Cache.Put(fmt.Sprint(userPlatformPrefix, userId), []byte(platform))
		if err != nil {
			c.Logger.Warn(fmt.Sprintf("save platform for %s error %v", userId, err))
		}
	} else {
		sharedKey = value
	}
	return sharedKey, nil
}
