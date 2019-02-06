package api

import (
	"encoding/json"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bridge"
	"go.uber.org/zap"
)

func init() {
	// legacy - use old slack name
	Router.Path("/api/v0/auth/team-tool").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var err error
			r, err = authorized(w, r)
			if err != nil {
				w.Write([]byte(`""`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			user, _ := bridge.Data.Slack.User(id)
			json.NewEncoder(w).Encode(bridge.OldEventToolAuthorization(user.Name))
		},
	)

	// V1 - uses member ID in the auth
	Router.Path("/api/v1/auth/team-tool").Methods("GET").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var err error
			r, err = authorized(w, r)
			if err != nil {
				w.Write([]byte(`""`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			member, err := DB.MemberBySlackID(id)
			if err != nil {
				Logger.Error("Unable to get member", zap.Error(err), zap.String("slackId", id))
			}
			json.NewEncoder(w).Encode(bridge.OldEventToolAuthorization(string(member.ID)))
		},
	)
}
