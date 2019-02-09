package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/FederationOfFathers/dashboard/config"
	"github.com/bwmarrin/discordgo"
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
	RedirectURL:  "https://dashboard.fofgaming.com/api/v1/oauth/discord",
}

func init() {
	initDiscordOauth()
}

func initDiscordOauth() {

	if config.DiscordConfig != nil {
		// Discord OAuth2
		conf.ClientID = config.DiscordConfig.ClientId
		conf.ClientSecret = config.DiscordConfig.Secret

		Router.Path("/api/v1/oauth/discord").Methods("GET").Handler(authenticated(discordOauthHandler))
	} else {
		Router.Path("/api/v1/oauth/discord").Methods("GET").Handler(authenticated(NotImplemented))
	}

}

func discordOauthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	query := r.URL.Query()
	code := query.Get("code")
	state := query.Get("state")

	if code == "" || state == "" {
		// if no code/state redirect to auth url
		authURL := conf.AuthCodeURL("asdasdasd13424yhion2f0") // TODO get proper state
		json.NewEncoder(w).Encode(authURL)
	} else {

		// exchange code for a user token
		ctx := context.Background()
		token, err := conf.Exchange(ctx, code)
		if err != nil {
			Logger.Error("Could not get token", zap.Error(err))
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

		// store the id to the db
		id := getMemberID(r)
		member, err := DB.MemberByID(id)
		if err != nil {
			Logger.Error("could not find member", zap.Int("member_id", id), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		member.Discord = userObj.ID
		member.Name = userObj.Username

		if err := member.Save(); err != nil {
			Logger.Error("unable to save discord id", zap.Int("member", member.ID), zap.String("discord id", userObj.ID), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// redirect
		http.Redirect(w, r, "https://ui.fofgaming.com/#main=members", http.StatusTemporaryRedirect)
	}

}
