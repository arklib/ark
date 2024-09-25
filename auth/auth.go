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

const StoreUserKey = "user"

var ErrAuthFailed = errx.New("auth failed", 401)

type Payload struct {
	*jwt.RegisteredClaims
}

type Auth struct {
	SecretKey []byte
	Expires   time.Duration
	// "header: Authorization, query: token, cookie: token"
	TokenLookup map[string]string
}

func New(secretKey []byte, expires time.Duration, tokenLookup string) (auth *Auth, err error) {
	if len(secretKey) == 0 {
		err = errx.Sprintf(
			"please configure auth secretKey: %s",
			NewAuthSecretKey())
		return
	}
	auth = &Auth{
		SecretKey:   secretKey,
		Expires:     expires,
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
	return
}

func (auth *Auth) NewToken(data map[string]any) (string, error) {
	expiresAt := time.Now().Add(auth.Expires)
	payload := &Payload{
		&jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString(auth.SecretKey)
}

func (auth *Auth) ParseToken(signed string) (user *User, err error) {
	payload := &Payload{}
	token, err := jwt.ParseWithClaims(
		signed,
		payload,
		func(token *jwt.Token) (any, error) {
			return auth.SecretKey, nil
		},
	)

	if err != nil || !token.Valid {
		err = ErrAuthFailed
	}

	return
}

func (auth *Auth) HttpMiddleware(roles ...string) hz.HandlerFunc {
	return func(ctx context.Context, req *hz.RequestContext) {
		token, err := auth.FindHttpToken(req)
		if err != nil {
			result.Error(req, err)
			return
		}

		user, err := auth.ParseToken(token)
		if err != nil || !lo.Contains(roles, user.Role) {
			result.Error(req, ErrAuthFailed)
			return
		}

		req.Set(StoreUserKey, user)
		req.Next(ctx)
	}
}

func (auth *Auth) FindHttpToken(req *hz.RequestContext) (token string, err error) {
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
			return
		}
	}

	err = ErrAuthFailed
	return
}

func NewAuthSecretKey() string {
	str := lo.RandomString(256, lo.AllCharset)
	secretKey := sha256.Sum256([]byte(str))
	return hex.EncodeToString(secretKey[:])
}
