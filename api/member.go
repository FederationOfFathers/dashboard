package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/uber-go/zap"
)

func init() {
	Router.Path("/api/v0/member/{memberID}").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			member, err := DB.MemberBySlackID(mux.Vars(r)["memberID"])
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err == gorm.ErrRecordNotFound || member == nil || member.Slack != mux.Vars(r)["memberID"] {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(member)
		},
	))

	Router.Path("/api/v0/member/{memberID}").Methods("PUT", "POST").Handler(
		jwtHandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				defer r.Body.Close()
				member, err := DB.MemberBySlackID(mux.Vars(r)["memberID"])
				if err != nil {
					http.NotFound(w, r)
					return
				}
				if member.Slack != mux.Vars(r)["memberID"] {
					http.NotFound(w, r)
					return
				}
				if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "json") {
					http.NotFound(w, r)
					return
				}
				var form = map[string]string{}

				err = json.NewDecoder(r.Body).Decode(&form)
				if err != nil {
					logger.Error("Error decoding JSON", zap.String("uri", r.URL.RawPath), zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				sid := getSlackUserID(r)
				admin, _ := bridge.Data.Slack.IsUserIDAdmin(sid)
				if sid != member.Slack && !admin {
					http.NotFound(w, r)
					return
				}

				changed := false
				for k, v := range form {
					switch strings.ToLower(k) {
					case "xbl":
						member.Xbl = v
						changed = true
					case "psn":
						member.Psn = v
						changed = true
					}
				}
				if changed {
					if err := member.Save(); err != nil {
						log.Println("Error saving member:", err.Error())
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
			},
		),
	)
}
