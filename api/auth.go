package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/uber-go/zap"
)

// AuthSecret is the secret used when generating mini auth tokens
var AuthSecret = ""

func init() {
	Router.Path("/api/v0/login").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if jwtSecretBytes == nil {
			jwtSecretBytes = []byte(JWTSecret)
		}
		args := r.URL.Query()
		who := args.Get("w")
		minitoken := args.Get("t")
		if validateMiniAuthToken(who, minitoken) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"userid": who,
			})
			tokenString, _ := token.SignedString(jwtSecretBytes)
			http.SetCookie(w, &http.Cookie{
				Name:     "Authorization",
				Value:    tokenString,
				Expires:  time.Now().Add(365 * 24 * time.Hour),
				HttpOnly: false,
				Path:     "/",
			})
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Invalid link. Please get another"))
	})
	Router.Path("/api/v0/logout").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})
}

func requireAdmin(w http.ResponseWriter, r *http.Request) error {
	id := getSlackUserID(r)
	if admin, err := slackData.IsUserIDAdmin(id); err != nil {
		logger.Error("error determining admin status", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return err
	} else if !admin {
		w.WriteHeader(http.StatusForbidden)
		return fmt.Errorf("error")
	}
	return nil
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
	// The loops in here don't break early on purpose.
	var valid = false
	var userValid = false
	for _, possible := range GenerateValidAuthTokens(forWhat) {
		if hmac.Equal([]byte(token), []byte(possible)) {
			valid = true
		}
	}
	for _, user := range slackData.Users {
		if user.ID == forWhat {
			userValid = true
		}
	}
	if valid == true && userValid == true {
		return true
	}
	return false
}

func getSlackUserID(r *http.Request) string {
	user := context.Get(r, "user").(*jwt.Token)
	if userid, ok := user.Claims.(jwt.MapClaims)["userid"]; ok {
		return userid.(string)
	}
	return ""
}
