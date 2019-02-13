package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FederationOfFathers/dashboard/db"
	"go.uber.org/zap"
)

type memberRestricted struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Slack   string `json:"slack"`
	Discord string `json:"discord"`
	Xbox    string `json:"xbox"`
}

func init() {
	Router.Path("/api/v1/members").Methods("GET").Handler(
		authenticated(
			func(w http.ResponseWriter, r *http.Request) {
				members, err := DB.Members()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					Logger.Error("Unable to retrieve members", zap.Error(err))
				}

				membersRestricted := membersToMembersRestricted(members)

				json.NewEncoder(w).Encode(membersRestricted)

			},
		))
}

func membersToMembersRestricted(members []*db.Member) map[string]memberRestricted {
	membersRestricted := map[string]memberRestricted{}
	for _, member := range members {
		membersRestricted[string(strconv.Itoa(member.ID))] = memberRestricted{
			ID:      member.ID,
			Name:    member.Name,
			Slack:   member.Slack,
			Discord: member.Discord,
			Xbox:    member.Xbl,
		}
	}

	return membersRestricted
}
