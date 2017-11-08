package api

import (
	"net/http"

	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
)

// JWTSecret is the Secret used when signing JTW tokens
var JWTSecret string

var jwtSecretBytes []byte

var jMW = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTSecret), nil
	},
	Extractor: func(r *http.Request) (string, error) {
		c, err := r.Cookie("Authorization")
		if err != nil {
			return "", err
		}
		return c.Value, nil
	},
	SigningMethod: jwt.SigningMethodHS256,
	ErrorHandler: func(w http.ResponseWriter, r *http.Request, err string) {
		Logger.Info(
			"HTTP Request",
			zap.String("uri", r.RequestURI),
			zap.Int("http_status", -1),
			zap.String("username", getSlackUserName(r)),
			zap.String("remote_address", r.RemoteAddr),
			zap.String("method", r.Method),
			zap.Int64("content_length", r.ContentLength),
			zap.String("error", err),
		)
		w.WriteHeader(http.StatusForbidden)
		w.Header().Set("Content-Type", "text/json")
		w.Write([]byte("{}"))
	},
})

func handlerFunc(fn func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(fn)
}

func jwtHandlerFunc(fn func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return jMW.Handler(http.HandlerFunc(fn))
}
