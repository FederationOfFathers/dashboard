package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/streams"
	"github.com/gorilla/mux"
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
			streamMemberID, err := strconv.Atoi(mux.Vars(r)["memberID"])
			if err != nil {
				logger.Error("Error converting memberID", zap.String("uri", r.URL.RawPath), zap.Error(err))
				http.NotFound(w, r)
				return
			}
			for _, stream := range streams.Streams {
				if stream.MemberID != streamMemberID {
					continue
				}
				json.NewEncoder(w).Encode(stream)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	Router.Path("/api/v0/streams").Methods("POST").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			kind := r.FormValue("kind")
			id := r.FormValue("id")
			userID := r.FormValue("userID")
			if userID == "" {
				userID = getSlackUserID(r)
			}
			err := streams.Add(kind, id, userID)
			if err != nil {
				logger.Error("Error adding stream", zap.String("uri", r.URL.RawPath), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
	))
}
