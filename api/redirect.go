package api

import (
	"net/http"

	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/bridge"
)

func init() {

	//TOD needed?
	Router.Path("/api/v1/redirect/team-tool").Methods("GET").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			id := getMemberID(r)
			user, _ := bot.Member(id)
			http.Redirect(w, r, bridge.OldEventToolLink(user.Nick), http.StatusTemporaryRedirect)
		},
	))
}
