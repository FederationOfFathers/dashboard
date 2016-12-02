package api

import (
	"encoding/json"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/gorilla/mux"
)

func init() {
	Router.Path("/api/v0/channels").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var channels = map[string]map[string]interface{}{}
			for _, channel := range bridge.Data.Slack.GetChannels() {
				members := []string{}
				for _, memberID := range channel.Members {
					user, err := bridge.Data.Slack.User(memberID)
					if err != nil {
						continue
					}
					members = append(members, user.Name)
				}
				channels[channel.ID] = map[string]interface{}{
					"id":      channel.ID,
					"name":    channel.Name,
					"topic":   channel.Topic.Value,
					"purpose": channel.Purpose.Value,
					"members": members,
					"visible": "true",
				}
			}
			json.NewEncoder(w).Encode(channels)
		},
	))

	Router.Path("/api/v0/channels/{channelID}/leave").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			want := mux.Vars(r)["channelID"]
			for _, channel := range bridge.Data.Slack.GetChannels() {
				if channel.ID == want {
					if err := bot.ChannelKick(want, id); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
					return
				}
			}
			http.NotFound(w, r)
		},
	))

	Router.Path("/api/v0/channels/{channelID}/join").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			want := mux.Vars(r)["channelID"]
			for _, channel := range bridge.Data.Slack.GetChannels() {
				if channel.ID == want {
					if err := bot.ChannelInvite(want, id); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
					return
				}
			}
			http.NotFound(w, r)
		},
	))

	Router.Path("/api/v0/channels/{channelID}/join").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			want := mux.Vars(r)["channelID"]
			for _, channel := range bridge.Data.Slack.GetChannels() {
				if channel.ID == want {
					if err := bot.ChannelInvite(want, id); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
					return
				}
			}
			http.NotFound(w, r)
		},
	))

}
