package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/FederationOfFathers/dashboard/config"
	"github.com/bwmarrin/discordgo"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

var discordEndpoint = oauth2.Endpoint{
	AuthURL:  "https://discordapp.com/api/oauth2/authorize",
	TokenURL: "https://discordapp.com/api/oauth2/token",
}

var conf = &oauth2.Config{
	ClientID:     "",
	ClientSecret: "",
	Scopes:       []string{"identify"},
	Endpoint:     discordEndpoint,
	RedirectURL:  "https://dashboard.fofgaming.com/api/v1/oauth/discord/verify",
}

func init() {
	initDiscordOauth()
}

func initDiscordOauth() {

	if config.DiscordConfig != nil {
		// Discord OAuth2
		conf.ClientID = config.DiscordConfig.ClientId
		conf.ClientSecret = config.DiscordConfig.Secret

		Router.Path("/api/v1/oauth/discord").Methods("GET").HandlerFunc(discordOauthHandler)
		Router.Path("/api/v1/oauth/discord/verify").Methods("GET").Handler(authenticated(discordOauthVerify))
		Router.Path("/api/v1/oauth/discord/login").Methods("GET").HandlerFunc(discordOauthVerify)
	} else {
		Router.Path("/api/v1/oauth/discord").Methods("GET").HandlerFunc(NotImplemented)
	}

}

func discordOauthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	authURL := conf.AuthCodeURL("asdasdasd13424yhion2f0") // TODO get proper state
	json.NewEncoder(w).Encode(authURL)

}

func discordOauthVerify(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	code := query.Get("code")
	state := query.Get("state")

	// get authenticated user
	id := getMemberID(r)

	// if id == 0, then this is a login, not a sync
	isAuthenticated := id != ""

	if code == "" || state == "" {
		w.WriteHeader(http.StatusBadRequest)
	} else {

		// exchange code for a user token
		ctx := context.Background()
		token, err := conf.Exchange(ctx, code)
		if err != nil {
			Logger.Error("Could not get token",
				zap.String("code", code),
				zap.String("state", state),
				zap.String("id", id),
				zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// create a new client with the token and get the user/@me endpoint
		client := conf.Client(ctx, token)
		res, err := client.Get("https://discordapp.com/api/users/@me")
		if err != nil {
			Logger.Error("Could not get user object", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// unmarshall the Body to a User{}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			Logger.Error("Could not parse body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		userObj := discordgo.User{}
		err = json.Unmarshal(body, &userObj)
		if err != nil {
			Logger.Error("Could not parse JSON", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// unauthenticated user
		if !isAuthenticated {
			member, err := DB.MemberByDiscordID(userObj.ID)
			if err == gorm.ErrRecordNotFound || err == sql.ErrNoRows {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("not authorizes"))
				return
			} else if err != nil {
				Logger.Error("unable to check member", zap.String("discordid", userObj.ID), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error"))
			}

			// set auth cookie and redirect
			authorize("", member.ID, w, r)

		} else {
			member, err := DB.MemberByAny(id)
			if err != nil {
				Logger.Error("could not find member", zap.String("member_id", id), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error"))
				return
			}

			// set discord id
			member.Discord = userObj.ID

			if err := member.Save(); err != nil {
				Logger.Error("unable to save discord id", zap.Int("member", member.ID), zap.String("discord id", userObj.ID), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error"))
				return
			}
		}

		// redirect
		http.Redirect(w, r, "https://ui.fofgaming.com/#main=members", http.StatusTemporaryRedirect)
	}

}
