package api

import (
	"encoding/json"
	"net/http"

	"github.com/FederationOfFathers/dashboard/slack"
	"github.com/gorilla/mux"
	"github.com/uber-go/zap"
	stow "gopkg.in/djherbis/stow.v2"
)

var visDB *stow.Store

func init() {
	Router.Path("/api/v0/groups").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			admin, _ := slackData.IsUserIDAdmin(id)
			var groups = map[string]map[string]interface{}{}
			for _, group := range slackData.GetGroups() {
				var visible string
				visDB.Get(group.ID, &visible)
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
					user, err := slackData.User(memberID)
					if err != nil {
						continue
					}
					members = append(members, user.Name)
				}
				groups[group.ID] = map[string]interface{}{
					"ID":      group.ID,
					"Name":    group.Name,
					"Members": members,
					"Visible": visible,
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
			visDB.Get(want, &visible)
			if visible == "true" {
				for _, group := range slackData.GetGroups() {
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
			err = visDB.Put(
				mux.Vars(r)["groupID"],
				r.FormValue("visible"))
			if err != nil {
				logger.Error("error putting a value to visDB", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
	))
}
