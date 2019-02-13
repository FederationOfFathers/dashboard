package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
)

func init() {
	// v0 using slack id
	Router.Path("/api/v0/member/{memberID}").Methods("GET").Handler(authenticated(
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

	// v1, using member id
	Router.Path("/api/v1/member/{memberID}").Methods("GET").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			// get memberId as an int, invalid member id returns a 404
			memberID, err := strconv.Atoi(mux.Vars(r)["memberID"])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("400 bad request"))
				return
			}

			// find the member
			member, err := DB.MemberByID(memberID)
			if err == gorm.ErrRecordNotFound || member == nil { // if not found, return 404
				http.NotFound(w, r)
				return
			} else if err != nil { // if db error, 500
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(member)
		},
	))

	// v0, using slack id
	Router.Path("/api/v0/member/{memberID}").Methods("PUT", "POST").Handler(
		authenticated(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				defer r.Body.Close()
				memberID := mux.Vars(r)["memberID"]
				member, err := DB.MemberBySlackID(memberID)
				if err != nil {
					Logger.Error("member lookup", zap.String("memberID", memberID), zap.Error(err))
					http.NotFound(w, r)
					return
				}
				if member.Slack != memberID {
					Logger.Error("member mismatch", zap.String("slack", member.Slack), zap.String("memberID", memberID), zap.Error(err))
					http.NotFound(w, r)
					return
				}
				if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "json") {
					Logger.Error("content-type", zap.Error(err))
					http.NotFound(w, r)
					return
				}
				var form = map[string]string{}

				err = json.NewDecoder(r.Body).Decode(&form)
				if err != nil {
					Logger.Error("Error decoding JSON", zap.String("uri", r.URL.RawPath), zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				sid := getSlackUserID(r)
				admin, _ := bridge.Data.Slack.IsUserIDAdmin(sid)
				if sid != member.Slack && !admin {
					http.NotFound(w, r)
					Logger.Debug(
						"access control",
						zap.String("sid", sid),
						zap.Bool("admin", admin),
						zap.String("slack", member.Slack))
					return
				}

				changed := false
				changedXBL := false
				for k, v := range form {
					switch strings.ToLower(k) {
					case "xbl":
						member.Xbl = v
						changed = true
						changedXBL = true
					case "psn":
						member.Psn = v
						changed = true
					}
				}
				if changed {
					if err := member.Save(); err != nil {
						Logger.Error("saving member", zap.Error(err))
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					if changedXBL {
						err := DB.Exec(
							strings.Join([]string{
								"INSERT INTO membermeta",
								"(member_id,meta_key,meta_value)",
								"VALUES (?,'_xbl_corrected',NOW())",
								"ON DUPLICATE KEY UPDATE meta_value=NOW()",
							}, " "),
							member.ID,
						).Error
						if err != nil {
							Logger.Error("error setting _xbl_corrected", zap.Int("member", member.ID), zap.Error(err))
						}

						err = DB.Exec("DELETE FROM membergames WHERE member = ?", member.ID).Error
						if err != nil {
							Logger.Error("error deleting membergames", zap.Int("member", member.ID), zap.Error(err))
						}
						err = DB.Exec(
							strings.Join([]string{
								"DELETE FROM membermeta",
								"WHERE member_id = ?",
								"AND meta_key IN(?,?,?)",
								"LIMIT 3",
							}, " "),
							member.ID,
							"_games_last_check",
							"_xuid_last_check",
							"xuid",
						).Error
						if err != nil {
							Logger.Error("error deleting membermeta", zap.Int("member", member.ID), zap.Error(err))
						}
					}
				}
			},
		),
	)
}
