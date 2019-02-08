package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/config"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

var discordEndpoint = oauth2.Endpoint{
	AuthURL:  "https://discordapp.com/api/oauth2/authorize",
	TokenURL: "https://discordapp.com/api/oauth2/token",
}

func init() {

	// Discord OAuth2
	conf := &oauth2.Config{
		ClientID:     config.DiscordConfig.ClientId,
		ClientSecret: config.DiscordConfig.Secret,
		Scopes:       []string{"identify"},
		Endpoint:     discordEndpoint,
		RedirectURL:  "https://dashboard.fofgaming.com/api/v1/oauth/discord",
	}

	Router.Path("/api/v1/oauth/discord").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			query := r.URL.Query()
			code := query.Get("code")
			state := query.Get("state")

			if code == "" || state == "" {
				// if no code/state redirect to auth url
				authURL := conf.AuthCodeURL("asdasdasd13424yhion2f0") // TODO get proper state
				http.Redirect(w, r, authURL, 302)
			} else {

				// exchange code for a user token
				ctx := context.Background()
				token, err := conf.Exchange(ctx, code)
				if err != nil {
					Logger.Error("Could not get token", zap.Error(err))
				}

				// create a new client with the token and get the user/@me endpoint
				client := conf.Client(ctx, token)
				res, err := client.Get("https://discordapp.com/api/user/@me")
				if err != nil {
					Logger.Error("Could not get user object", zap.Error(err))
				}

				// unmarshall the Body to a User{}
				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					Logger.Error("Could not parse body", zap.Error(err))
				}
				userObj := discordgo.User{}
				err = json.Unmarshal(body, &userObj)
				if err != nil {
					Logger.Error("Could not parse JSON", zap.Error(err))
				}

				// store the id to the db
				id := getMemberID(r)
				member, err := DB.MemberByID(id)
				if err != nil {
					Logger.Error("could not find member", zap.Int("member_id", id), zap.Error(err))
				}
				member.Discord = userObj.ID
				member.Save()

			}

		},
	)
}

func loadDiscordCfg() *bot.DiscordCfg {
	return &bot.DiscordCfg{}
}
