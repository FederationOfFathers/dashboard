package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/streams"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/uber-go/zap"
)

func init() {
	Router.Path("/api/v0/streams").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			enc := json.NewEncoder(w)
			enc.Encode(streams.Streams)
		},
	))

	Router.Path("/api/v0/streams/{memberID}/{type}").Methods("DELETE").Handler(jwtHandlerFunc(
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
			stream, err := DB.StreamByMemberID(member.ID)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if err := streams.Remove(stream.ID, mux.Vars(r)["type"]); err != nil {
				logger.Error("Error removing stream", zap.String("uri", r.URL.RawPath), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
	))

	Router.Path("/api/v0/streams/{memberID}").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			member, err := DB.MemberByAny(mux.Vars(r)["memberID"])
			if err == gorm.ErrRecordNotFound {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			stream, err := DB.StreamByMemberID(member.ID)
			if err == gorm.ErrRecordNotFound {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(stream)
		},
	))

	streamSetHandler := jwtHandlerFunc(
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
					logger.Error("Error decoding JSON", zap.String("uri", r.URL.RawPath), zap.Error(err))
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
				userID = getSlackUserID(r)
			}
			err := streams.Add(kind, id, userID)
			if err != nil {
				logger.Error(
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
	Router.Path("/api/v0/streams").Methods("PUT").Handler(streamSetHandler)
	Router.Path("/api/v0/streams").Methods("POST").Handler(streamSetHandler)
}
