package api

import (
	"fmt"
	"net/http"
)

func init() {
	Router.Path("/api/v0/ping").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			id := getSlackUserID(r)
			user, _ := slackData.User(id)
			fmt.Fprintf(w, "This is an authenticated request\n\n\n")
			fmt.Fprintf(w, "user: %#v\n\n\n", user)
			fmt.Fprintf(w, "groups: %#v\n\n\n", slackData.UserGroups(id))
			fmt.Fprintf(w, "channels: %#v", slackData.UserChannels(id))
		},
	))
}
