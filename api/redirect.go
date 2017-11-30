package api

import (
	"net/http"

	"github.com/FederationOfFathers/dashboard/bridge"
)

func init() {
	Router.Path("/api/v0/redirect/team-tool").Methods("GET").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			id := getSlackUserID(r)
			user, _ := bridge.Data.Slack.User(id)
			http.Redirect(w, r, bridge.OldEventToolLink(user.Name), http.StatusTemporaryRedirect)
		},
	))
}
