package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/auth0/go-jwt-middleware"
	"github.com/codegangsta/negroni"
	"github.com/dgrijalva/jwt-go"
)

// JWTSecret is the Secret used when signing JTW tokens
var JWTSecret string

// AuthSecret is the secret used when generating mini auth tokens
var AuthSecret = ""

func init() {
	Router.Path("/api/v0/login").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		args := r.URL.Query()
		who := args.Get("w")
		token := args.Get("t")
		fmt.Fprintln(w, validateMiniAuthToken(who, token))
	})
	Router.Path("/api/v0/logout").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})
}

// GenerateValidAuthTokens generates all possible valid auth tokens for right now.
// To me used both when vreating new tokens and validating incoming tokens
func GenerateValidAuthTokens(forWhat string) []string {
	var now = int(time.Now().Unix() / 300)
	return []string{
		generateMiniAuthToken(forWhat, now),
		generateMiniAuthToken(forWhat, now-1),
	}
}

func generateMiniAuthToken(forWhat string, when int) string {
	mac := hmac.New(sha256.New, []byte(AuthSecret))
	mac.Write([]byte(fmt.Sprintf(":%s:%d:", forWhat, when)))
	return fmt.Sprintf("%0x", mac.Sum(nil))[22:34]
}

func validateMiniAuthToken(forWhat, token string) bool {
	var valid = false
	for _, possible := range GenerateValidAuthTokens(forWhat) {
		if hmac.Equal([]byte(token), []byte(possible)) {
			valid = true
		}
	}
	return valid
}

var jMW = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		log.Println(JWTSecret)
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

func jwtHandlerFunc(fn func(w http.ResponseWriter, r *http.Request)) *negroni.Negroni {
	return negroni.New(
		negroni.HandlerFunc(jMW.HandlerWithNext),
		negroni.Wrap(http.HandlerFunc(fn)),
	)
}
