package api

import (
	"encoding/json"
	"net/http"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/streams"
	"github.com/gorilla/mux"
)

func init() {
	Router.Path("/api/v0/streams").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			enc := json.NewEncoder(w)
			enc.Encode(streams.Streams)
		},
	))

	Router.Path("/api/v0/streams/{key}").Methods("DELETE").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := getSlackUserID(r)
			admin, _ := bridge.Data.Slack.IsUserIDAdmin(id)
			if stream, ok := streams.Streams[mux.Vars(r)["key"]]; ok {
				if stream.UserID == id || admin {
					if err := streams.Remove(stream.Kind, stream.ServiceID); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
				}
				return
			}
			w.WriteHeader(http.StatusNotFound)
		},
	))

	Router.Path("/api/v0/streams/{key}").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if stream, ok := streams.Streams[mux.Vars(r)["key"]]; ok {
				enc := json.NewEncoder(w)
				enc.Encode(stream)
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
				w.WriteHeader(http.StatusInternalServerError)
			}
		},
	))
}
