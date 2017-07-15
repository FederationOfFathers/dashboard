package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/uber-go/zap"
)

func init() {
	Router.Path("/api/v0/meta/member/{memberID}/{key}").Methods("DELETE").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			admin, _ := bridge.Data.Slack.IsUserIDAdmin(id)
			member, err := DB.MemberBySlackID(mux.Vars(r)["memberID"])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if strings.ToLower(member.Slack) != strings.ToLower(id) && !admin {
				http.NotFound(w, r)
				return
			}
			DB.Delete(db.MemberMeta{}, "member_ID = ? AND meta_key = ?", member.ID, mux.Vars(r)["key"])
			w.Write([]byte("{}"))
		},
	))

	Router.Path("/api/v0/meta/member/{memberID}").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			member, err := DB.MemberByAny(mux.Vars(r)["memberID"])
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err == gorm.ErrRecordNotFound || member == nil {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var out = map[string]string{}
			var entries []*db.MemberMeta
			DB.Where("member_ID = ?", member.ID).Find(&entries)
			for _, entry := range entries {
				out[entry.MetaKey] = string(entry.MetaJSON)
			}
			json.NewEncoder(w).Encode(out)
		},
	))

	Router.Path("/api/v0/meta/member/{memberID}/{key}").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			member, err := DB.MemberByAny(mux.Vars(r)["memberID"])
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err == gorm.ErrRecordNotFound || member == nil {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var out = map[string]string{}
			var entries []*db.MemberMeta
			DB.Where("member_ID = ? AND meta_key = ?", member.ID, mux.Vars(r)["key"]).Find(&entries)
			for _, entry := range entries {
				out[entry.MetaKey] = string(entry.MetaJSON)
			}
			json.NewEncoder(w).Encode(out)
		},
	))

	Router.Path("/api/v0/meta/member/{memberID}").Methods("PUT", "POST").Handler(
		jwtHandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				defer r.Body.Close()
				member, err := DB.MemberBySlackID(mux.Vars(r)["memberID"])
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
					logger.Error("Error decoding JSON", zap.String("uri", r.URL.RawPath), zap.Error(err))
				}

				sid := getSlackUserID(r)
				admin, _ := bridge.Data.Slack.IsUserIDAdmin(sid)
				if sid != member.Slack && !admin {
					http.NotFound(w, r)
					return
				}

				for k, v := range form {
					err := DB.Exec(
						"INSERT INTO member_meta (`member_id`,`meta_key`,`meta_json`,`created_at`,`updated_at`) "+
							"VALUES(?,?,?,NOW(),NOW()) "+
							"ON DUPLICATE KEY UPDATE "+
							"`meta_json` = ?, `updated_at` = NOW(), `deleted_at` = NULL",
						member.ID,
						k,
						[]byte(v),
						[]byte(v),
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
