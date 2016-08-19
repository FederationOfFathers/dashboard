package api

import (
	"encoding/json"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/store"
	"github.com/gorilla/mux"
	"github.com/uber-go/zap"
	stow "gopkg.in/djherbis/stow.v2"
)

func visDB() *stow.Store {
	return store.DB.Groups().NewNestedStore([]byte("visibility-v1"))
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
			if visible == "true" {
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

	Router.Path("/api/v0/groups/{groupID}/visibility").Methods("PUT").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			err := requireAdmin(w, r)
			if err != nil {
				return
			}
			err = visDB().Put(
				mux.Vars(r)["groupID"],
				r.FormValue("visible"))
			if err != nil {
				logger.Error("error putting a value to visDB", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
	))
}
