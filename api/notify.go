package api

import (
	"net/http"

	"github.com/FederationOfFathers/dashboard/bridge"
)

func init() {
	Router.Path("/api/v0/notify/slack-core-data").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			bridge.SlackCoreDataUpdated.L.Lock()
			bridge.SlackCoreDataUpdated.Wait()
		},
	))
}
