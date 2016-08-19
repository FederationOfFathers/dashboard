package api

import (
	"encoding/json"
	"net/http"

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
