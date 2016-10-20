package api

import (
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/auth0/go-jwt-middleware"
	"github.com/codegangsta/negroni"
	"github.com/dgrijalva/jwt-go"
	"github.com/rabeesh/negroni-nocache"
	"github.com/rs/cors"
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
})

func handlerFunc(fn func(w http.ResponseWriter, r *http.Request)) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS", "DELETE"},
		AllowedHeaders:   []string{"Content-Type"},
	})
	return gziphandler.GzipHandler(
		c.Handler(
			negroni.New(
				&httpLogger{},
				negroni.NewRecovery(),
				nocache.New(true),
				negroni.Wrap(
					http.HandlerFunc(fn),
				),
			),
		),
	)
}

func jwtHandlerFunc(fn func(w http.ResponseWriter, r *http.Request)) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "OPTIONS", "DELETE"},
		AllowedHeaders:   []string{"Content-Type"},
	})
	return gziphandler.GzipHandler(
		c.Handler(
			jMW.Handler(
				negroni.New(
					&httpLogger{},
					negroni.NewRecovery(),
					nocache.New(true),
					negroni.Wrap(
						http.HandlerFunc(fn),
					),
				),
			),
		),
	)
}
