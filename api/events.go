package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
)

func init() {
	Router.Path("/api/v0/events").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			events, err := DB.Events()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(events)
		},
	))

	Router.Path("/api/v0/events").Methods("POST").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			timestamp, err := strconv.Atoi(r.FormValue("when"))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			member, err := DB.MemberBySlackID(getSlackUserID(r))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			event := DB.NewEvent()
			event.Where = r.FormValue("where")
			event.Title = r.FormValue("title")
			if t := time.Unix(int64(timestamp), 0); true {
				event.When = &t
			}
			event.Members = []db.EventMember{
				{Member: *member},
			}
			if err := event.Save(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(event)
		},
	))
}
