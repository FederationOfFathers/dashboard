package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/streams"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
)

func init() {
	Router.Path("/api/v0/streams").Methods("GET").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			enc := json.NewEncoder(w)
			enc.Encode(streams.Streams)
		},
	))

	Router.Path("/api/v1/streams/{memberID}/{type}").Methods("DELETE").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getMemberID(r)
			m, err := DB.MemberByAny(id)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			admin, _ := bot.IsUserIDAdmin(m.Discord)
			member, err := DB.MemberByAny(mux.Vars(r)["memberID"])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if strings.ToLower(member.Slack) != strings.ToLower(id) && !admin {
				http.NotFound(w, r)
				return
			}
			stream, err := DB.StreamByMemberID(member.ID)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if err := streams.Remove(stream.ID, mux.Vars(r)["type"]); err != nil {
				Logger.Error("Error removing stream", zap.String("uri", r.URL.RawPath), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
	))

	Router.Path("/api/v1/streams/{memberID}").Methods("GET").Handler(authenticated(
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
			stream, err := DB.StreamByMemberID(member.ID)
			if err != nil && err != gorm.ErrRecordNotFound {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(stream)
		},
	))

	streamSetHandler := authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			var kind string
			var id string
			var userID string
			if strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "json") {
				var form = struct {
					Kind   string `json:"kind"`
					ID     string `json:"id"`
					UserID string `json:"userID"`
				}{}
				err := json.NewDecoder(r.Body).Decode(&form)
				if err != nil {
					Logger.Error("Error decoding JSON", zap.String("uri", r.URL.RawPath), zap.Error(err))
				}
				kind = form.Kind
				id = form.ID
				userID = form.UserID
			} else {
				kind = r.FormValue("kind")
				id = r.FormValue("id")
				userID = r.FormValue("userID")
			}
			w.Header().Set("Content-Type", "application/json")
			if userID == "" {
				userID = getMemberID(r)
			}

			mid := getMemberID(r)
			m, merr := DB.MemberByAny(mid)
			if merr != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			admin, _ := bot.IsUserIDAdmin(m.Discord)
			if mid != userID {
				if !admin {
					http.NotFound(w, r)
					return
				}
			}

			err := streams.Add(kind, id, userID)
			if err != nil {
				Logger.Error(
					"Error adding stream",
					zap.String("uri", r.URL.RawPath),
					zap.String("kind", kind),
					zap.String("id", id),
					zap.String("userID", userID),
					zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
	)
	Router.Path("/api/v1/streams").Methods("PUT").Handler(streamSetHandler)
	Router.Path("/api/v1/streams").Methods("POST").Handler(streamSetHandler)
}
