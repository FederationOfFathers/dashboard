package api

import (
	"encoding/json"
	"net/http"
)

func init() {
	Router.Path("/api/v0/ping").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			user, _ := slackData.User(id)
			enc := json.NewEncoder(w)
			enc.Encode(map[string]interface{}{
				"user":     user,
				"groups":   slackData.UserGroups(id),
				"channels": slackData.UserChannels(id),
			})
		},
	))
}
