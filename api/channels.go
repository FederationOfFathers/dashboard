package api

import (
	"encoding/json"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/gorilla/mux"
)

func init() {

	Router.Path("/api/v1/channels").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set("Content-Type", "application/json")

			var response = struct {
				Users    []string                          `json:"users"`
				Channels map[string]map[string]interface{} `json:"channels"`
			}{
				[]string{},
				map[string]map[string]interface{}{},
			}

			var lookup = map[string]int{}
			for _, user := range bridge.Data.Slack.GetUsers() {
				lookup[user.ID] = len(response.Users)
				response.Users = append(response.Users, user.Name)
			}

			for _, channel := range bridge.Data.Slack.GetChannels() {
				members := []int{}
				for _, memberID := range channel.Members {
					members = append(members, lookup[memberID])
				}
				response.Channels[channel.ID] = map[string]interface{}{
					"id":      channel.ID,
					"name":    channel.Name,
					"topic":   channel.Topic.Value,
					"purpose": channel.Purpose.Value,
					"members": members,
					"visible": "true",
				}
			}
			json.NewEncoder(w).Encode(response)
		},
	))

	Router.Path("/api/v0/channels").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var lookup = map[string]string{}
			for _, user := range bridge.Data.Slack.GetUsers() {
				lookup[user.ID] = user.Name
			}

			w.Header().Set("Content-Type", "application/json")
			var channels = map[string]map[string]interface{}{}
			for _, channel := range bridge.Data.Slack.GetChannels() {
				members := []string{}
				for _, memberID := range channel.Members {
					members = append(members, lookup[memberID])
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
