package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
	"go.uber.org/zap"
)

func init() {
	Router.Path("/api/v1/events").Methods("GET").Handler(authenticated(
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

	// Create an event, needs when (time), where(channel id), title, and member from request
	Router.Path("/api/v1/events/create").Methods("POST").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			// convert time
			timestamp, err := strconv.Atoi(r.FormValue("when"))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// get the member
			member, err := DB.MemberByID(getMemberID(r))

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// get channel in DB
			where, err := strconv.Atoi(r.FormValue("where"))
			if err != nil {
				Logger.Error("invalid channel id", zap.String("where", r.FormValue("where")), zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			eventChannel, err := DB.EventChannelByID(where)
			if err != nil {
				Logger.Error("Invalid event channel", zap.Int("channel_id", where), zap.Error(err))
			}

			// build the event
			event := DB.NewEvent()
			event.EventChannel = *eventChannel
			event.Title = r.FormValue("title")
			event.Members = []db.EventMember{
				{Member: *member},
			}
			if t := time.Unix(int64(timestamp), 0); true {
				event.When = &t
			}

			// save the event
			if err := event.Save(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(event)
		},
	))

	// Join an event
	Router.Path("/api/v1/events/join").Methods("POST").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			member, err := DB.MemberByID(getMemberID(r))
			if err != nil {
				Logger.Error("could not get a valid member", zap.Error(err))
			}
			eventID, err := strconv.Atoi(r.FormValue("eventID"))
			if err != nil {
				Logger.Error("bad event id", zap.String("eventID", r.FormValue("eventID")), zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			event, err := DB.EventByID(eventID)

			event.Members = append(event.Members, db.EventMember{Member: *member})
			event.Save()

			w.WriteHeader(http.StatusOK)

		},
	))

	Router.Path("/api/v1/events/channels").Methods("GET").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			eventChannels, err := DB.EventChannels()
			if err != nil {
				Logger.Error("unable to get event channels", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}

			json.NewEncoder(w).Encode(eventChannels)
		},
	))
}
