package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	hz "github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"
	"github.com/samber/lo"

	"github.com/arklib/ark/errx"
	"github.com/arklib/ark/http/result"
)

const StoreAuthKey = "auth"

var ErrAuthFailed = errx.New("auth failed", 401)

type Payload = jwt.MapClaims

type Auth struct {
	SecretKey   []byte
	Expire      int64
	TokenLookup map[string]string
}

// tokenLookup: "header: Authorization, query: token, cookie: token"
func New(secretKey string, expire int64, tokenLookup string) (*Auth, error) {
	if len(secretKey) == 0 {
		err := errx.Sprintf(
			"please configure auth secretKey: %s",
			NewAuthSecretKey())
		return nil, err
	}
	auth := &Auth{
		SecretKey:   []byte(secretKey),
		Expire:      expire,
		TokenLookup: make(map[string]string),
	}

	// trim space
	tokenLookup = strings.ReplaceAll(tokenLookup, " ", "")
	for _, lookup := range strings.Split(tokenLookup, ",") {
		parts := strings.SplitN(lookup, ":", 2)
		if len(parts) != 2 {
			continue
		}
		method, name := parts[0], parts[1]
		auth.TokenLookup[method] = name
	}
	return auth, nil
}

func (auth *Auth) NewToken(scene string, payload Payload) (string, error) {
	claims := make(jwt.MapClaims)
	for key, value := range payload {
		claims[key] = value
	}

	now := time.Now().Unix()
	claims["exp"] = now + auth.Expire
	claims["iat"] = now
	claims["scene"] = scene

	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	return token.SignedString(auth.SecretKey)
}

func (auth *Auth) ParseToken(signed string) (Payload, error) {
	claims := make(jwt.MapClaims)
	token, err := jwt.ParseWithClaims(
		signed,
		claims,
		func(token *jwt.Token) (any, error) {
			return auth.SecretKey, nil
		},
	)

	if err != nil || !token.Valid {
		return nil, ErrAuthFailed
	}
	return claims, nil
}

func (auth *Auth) HttpMiddleware(scenes ...string) hz.HandlerFunc {
	return func(ctx context.Context, req *hz.RequestContext) {
		token, err := auth.FindHttpToken(req)
		if err != nil {
			result.Error(req, err)
			return
		}

		payload, err := auth.ParseToken(token)
		if err != nil {
			result.Error(req, err)
			return
		}

		if len(scenes) > 0 {
			scene, ok := payload["scene"]
			if !ok || !lo.Contains(scenes, scene.(string)) {
				result.Error(req, ErrAuthFailed)
				return
			}
		}
		req.Set(StoreAuthKey, payload)
		req.Next(ctx)
	}
}

func (auth *Auth) FindHttpToken(req *hz.RequestContext) (string, error) {
	token := ""
	for method, name := range auth.TokenLookup {
		switch method {
		case "header":
			bearer := string(req.GetHeader(name))
			if strings.HasPrefix(bearer, "Bearer ") {
				token = bearer[7:]
			}
		case "cookie":
			token = string(req.Cookie(name))
		case "query":
			token = req.Query(name)
		}

		if len(token) > 0 {
			return token, nil
		}
	}

	return "", ErrAuthFailed
}

func NewAuthSecretKey() string {
	str := lo.RandomString(256, lo.AllCharset)
	secretKey := sha256.Sum256([]byte(str))
	return hex.EncodeToString(secretKey[:])
}
