package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
)

func init() {
	Router.Path("/api/v1/meta/member/{memberID}/{key}").Methods("DELETE").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getMemberID(r)
			authMember, err := DB.MemberByAny(id)
			if err != nil {
				Logger.Error("invalid member", zap.String("id", id), zap.Error(err))
				w.WriteHeader(http.StatusForbidden)
				return
			}
			admin, _ := bot.IsUserIDAdmin(authMember.Discord)
			member, err := DB.MemberByAny(mux.Vars(r)["memberID"])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if strings.ToLower(member.Slack) != strings.ToLower(id) && !admin {
				http.NotFound(w, r)
				return
			}
			DB.Delete(db.MemberMeta{}, "member_ID = ? AND meta_key = ?", member.ID, mux.Vars(r)["key"])
		},
	))

	Router.Path("/api/v1/meta/member/{memberID}").Methods("GET").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			member, err := DB.MemberByAny(mux.Vars(r)["memberID"])
			if err == gorm.ErrRecordNotFound || member == nil {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				Logger.Error("querying user", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var out = map[string]string{}
			rows, err := DB.Raw("SELECT meta_key,meta_value FROM membermeta WHERE member_id = ?", member.ID).Rows()
			defer rows.Close()
			if err != nil {
				Logger.Error("querying", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			for rows.Next() {
				var k string
				var v string
				if err := rows.Scan(&k, &v); err != nil {
					Logger.Error("scanning", zap.Error(err))
					continue
				}
				out[k] = v
			}
			json.NewEncoder(w).Encode(out)
		},
	))

	Router.Path("/api/v1/meta/member/{memberID}").Methods("PUT", "POST").Handler(
		authenticated(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				defer r.Body.Close()
				member, err := DB.MemberByAny(mux.Vars(r)["memberID"])
				if err != nil {
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
					Logger.Error("Error decoding JSON", zap.String("uri", r.URL.RawPath), zap.Error(err))
				}

				id := getMemberID(r)
				m, err := DB.MemberByAny(id)
				if err != nil {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				admin, _ := bot.IsUserIDAdmin(m.Discord)
				if id != member.Discord && !admin {
					http.NotFound(w, r)
					return
				}

				for k, v := range form {
					err := DB.Exec(strings.Join([]string{
						"INSERT INTO membermeta (`member_id`,`meta_key`,`meta_value`)",
						"VALUES(?,?,?)",
						"ON DUPLICATE KEY UPDATE",
						"`meta_value` = ?",
					}, " "),
						member.ID,
						k,
						v,
						v,
					).Error
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
			},
		),
	)
}
