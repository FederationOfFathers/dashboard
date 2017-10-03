package api

import (
	"encoding/json"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bridge"
)

func init() {
	Router.Path("/api/v0/auth/team-tool").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			user, _ := bridge.Data.Slack.User(id)
			json.NewEncoder(w).Encode(bridge.OldEventToolAuthorization(user.Name))
		},
	))
}
