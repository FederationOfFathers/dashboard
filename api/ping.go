package api

import (
	"encoding/json"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bridge"
)

func init() {
	Router.Path("/api/v0/ping").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			user, _ := bridge.Data.Slack.User(id)
			admin, _ := bridge.Data.Slack.IsUserIDAdmin(id)
			enc := json.NewEncoder(w)
			enc.Encode(map[string]interface{}{
				"user":     user,
				"admin":    admin,
				"groups":   bridge.Data.Slack.UserGroups(id),
				"channels": bridge.Data.Slack.UserChannels(id),
			})
		},
	))
}
