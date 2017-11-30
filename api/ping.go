package api

import (
	"encoding/json"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bridge"
)

func init() {
	Router.Path("/api/v0/ping").Methods("GET").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			w.Header().Set("X-UID", id)
			user, _ := bridge.Data.Slack.User(id)
			admin, _ := bridge.Data.Slack.IsUserIDAdmin(id)
			userGroups := bridge.Data.Slack.UserGroups(id)
			userGroupsVisible := map[string]string{}
			for _, group := range userGroups {
				var visible string
				visDB().Get(group.ID, &visible)
				if visible != "true" {
					visible = "false"
				}
				userGroupsVisible[group.ID] = visible
			}
			var rval = map[string]interface{}{
				"user":          user,
				"admin":         admin,
				"groups":        userGroups,
				"group_visible": userGroupsVisible,
				"channels":      bridge.Data.Slack.UserChannels(id),
			}
			json.NewEncoder(w).Encode(rval)
		},
	))
}
