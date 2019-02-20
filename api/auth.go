package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/FederationOfFathers/dashboard/bot"
	"go.uber.org/zap"
)

// AuthSecret is the secret used when generating mini auth tokens
var AuthSecret = ""

func init() {

	// v0 using slack
	Router.Path("/api/v0/login").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		args := r.URL.Query()
		who := args.Get("w")
		minitoken := args.Get("t")
		if validateMiniAuthToken(who, minitoken) {
			authorize(who, 0, w, r)
			if args.Get("r") == "0" {
				w.Header().Set("Content-Type", "text/json")
				json.NewEncoder(w).Encode("ok")
				return
			}
			http.Redirect(w, r, "https://ui.fofgaming.com/", http.StatusTemporaryRedirect)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Invalid link. Please get another"))
	})

	// v1 using member id
	Router.Path("/api/v1/login").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		args := r.URL.Query()
		who, err := strconv.Atoi(args.Get("w")) // if this conversion fails, then we have a bad request
		minitoken := args.Get("t")
		if err == nil && validateMiniAuthTokenForID(who, minitoken) {
			authorize("", who, w, r)
			if args.Get("r") == "0" {
				w.Header().Set("Content-Type", "text/json")
				json.NewEncoder(w).Encode("ok")
				return
			}
			http.Redirect(w, r, "https://ui.fofgaming.com/", http.StatusTemporaryRedirect)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Invalid link. Please get another"))
	})
	Router.Path("/api/v0/logout").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Expires:  time.Now().Add(0 - (365 * 24 * time.Hour)),
			HttpOnly: false,
			Path:     "/",
		})
	})
}

func requireAdmin(w http.ResponseWriter, r *http.Request) error {
	id := getMemberID(r)
	member, err := DB.MemberByDiscordID(id)
	if err != nil {
		Logger.Error("could not find this user", zap.String("id", id), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	if admin, err := bot.IsUserIDAdmin(member.Discord); err != nil {
		Logger.Error("error determining admin status", zap.Error(err))
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

// Deprecated for slack usage
func validateMiniAuthToken(forWhat, token string) bool {
	// The loops in here don't break early on purpose.
	var valid = false
	var userValid = false
	for _, possible := range GenerateValidAuthTokens(forWhat) {
		if hmac.Equal([]byte(token), []byte(possible)) {
			valid = true
		}
	}
	for _, user := range bot.GetMembers() {
		if user.User.ID == forWhat {
			userValid = true
		}
	}

	if valid == true && userValid == true {
		return true
	}
	return false
}

// runs validation for member id
func validateMiniAuthTokenForID(forID int, token string) bool {
	// The loops in here don't break early on purpose.
	var valid = false
	var userValid = false
	for _, possible := range GenerateValidAuthTokens(strconv.Itoa(forID)) {
		if hmac.Equal([]byte(token), []byte(possible)) {
			valid = true
		}
	}

	member, err := DB.MemberByID(forID)
	if err != nil || member.ID <= 0 {
		userValid = false
	}

	userValid = member.ID == forID

	if valid == true && userValid == true {
		return true
	}
	return false
}

func getSlackUserID(r *http.Request) string {
	auth := requestAuth(r)
	id := auth["userid"]
	if id != "" {
		return id
	}

	// if no userid, look for memberid and find the slackid. return '' on error
	memberidStr := auth["memberid"]
	if memberidStr != "" {
		if memberid, err := strconv.Atoi(memberidStr); err == nil {
			member, err := DB.MemberByID(memberid)
			if err == nil {
				return member.Slack
			}
		}
	}

	return id
}

func getMemberID(r *http.Request) string {
	auth := requestAuth(r)
	id := auth["memberid"]
	if id != "" {
		return id
	}

	// if no memberid, try slack id
	slackid := auth["userid"]
	return slackid

}

func getSlackUserName(r *http.Request) string {
	member, err := DB.MemberBySlackID(getSlackUserID(r))
	if err != nil || member == nil {
		return ""
	}
	return member.Name
}
