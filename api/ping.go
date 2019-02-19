package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bot"
)

func init() {

	Router.Path("/api/v1/ping").Methods("GET").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getMemberID(r)
			Logger.Debug(fmt.Sprintf("id: %s", id))
			w.Header().Set("X-UID", id)
			member, _ := DB.MemberByAny(id)
			admin, _ := bot.IsUserIDAdmin(member.Discord)
			dMember, _ := bot.Member(member.Discord)
			var rval = map[string]interface{}{
				"user":   member,
				"member": dMember,
				"admin":  admin,
			}
			json.NewEncoder(w).Encode(rval)
		},
	))
}
