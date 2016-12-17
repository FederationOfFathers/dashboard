package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/store"
	"github.com/gorilla/mux"
	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
	stow "gopkg.in/djherbis/stow.v2"
)

func visDB() *stow.Store {
	return store.DB.Groups().NewNestedStore([]byte("visibility-v1"))
}

func isSlackUserInGroup(slackID, groupID string) bool {
	for _, group := range bridge.Data.Slack.GetGroups() {
		if group.ID != groupID {
			continue
		}
		for _, member := range group.Members {
			if member == slackID {
				return true
			}
		}
		return false
	}
	return false
}

func init() {
	Router.Path("/api/v0/groups").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			admin, _ := bridge.Data.Slack.IsUserIDAdmin(id)
			var groups = map[string]map[string]interface{}{}
			for _, group := range bridge.Data.Slack.GetGroups() {
				var visible string
				visDB().Get(group.ID, &visible)
				if visible != "true" {
					visible = "false"
				}
				if visible == "false" {
					if !admin {
						continue
					}
				}
				members := []string{}
				for _, memberID := range group.Members {
					user, err := bridge.Data.Slack.User(memberID)
					if err != nil {
						continue
					}
					members = append(members, user.Name)
				}
				groups[group.ID] = map[string]interface{}{
					"id":      group.ID,
					"name":    group.Name,
					"topic":   group.Topic.Value,
					"purpose": group.Purpose.Value,
					"members": members,
					"visible": visible,
				}
			}
			enc := json.NewEncoder(w)
			enc.Encode(groups)
		},
	))

	Router.Path("/api/v0/groups/{groupID}/join").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			want := mux.Vars(r)["groupID"]
			var visible string
			visDB().Get(want, &visible)
			if visible == "true" || isSlackUserInGroup(id, want) {
				for _, group := range bridge.Data.Slack.GetGroups() {
					if group.ID != want {
						continue
					}
					if err := bot.GroupInvite(group.ID, id); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	Router.Path("/api/v0/groups/{groupID}/leave").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			want := mux.Vars(r)["groupID"]
			var visible string
			visDB().Get(want, &visible)
			if visible == "true" || isSlackUserInGroup(id, want) {
				for _, group := range bridge.Data.Slack.GetGroups() {
					if group.ID != want {
						continue
					}
					if err := bot.GroupKick(group.ID, id); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	Router.Path("/api/v0/groups/{groupID}/visibility").Methods("PUT", "POST", "OPTIONS").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if r.Method == "OPTIONS" {
				return
			}

			var group *slack.Group

			w.Header().Set("Content-Type", "application/json")

			for _, g := range bridge.Data.Slack.GetGroups() {
				if mux.Vars(r)["groupID"] == g.ID {
					group = &g
					break
				}
			}

			if group == nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			id := getSlackUserID(r)
			member, err := DB.MemberBySlackID(id)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if err := requireAdmin(w, r); err != nil {
				// Not an admin... so a request
				bot.SendMessage(
					"damnbot-admin",
					fmt.Sprintf(
						"Request to change group visibility for *%s* received from *%s*",
						group.Name,
						member.Name,
					),
				)
				return
			}

			var doc = struct {
				Visibility string `json:"visible"`
			}{}

			if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
				logger.Error("Failed decoding json for group visibility change", zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if err = visDB().Put(mux.Vars(r)["groupID"], doc.Visibility); err != nil {
				logger.Error("error putting a value to visDB", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
	))
}
