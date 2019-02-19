package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FederationOfFathers/dashboard/bridge"
	"go.uber.org/zap"
)

func init() {

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
			id := getMemberID(r)
			member, err := DB.MemberByAny(id)
			if err != nil {
				Logger.Error("Unable to get member", zap.Error(err), zap.String("slackId", id))
			}
			json.NewEncoder(w).Encode(bridge.OldEventToolAuthorization(strconv.Itoa(member.ID)))
		},
	)
}
