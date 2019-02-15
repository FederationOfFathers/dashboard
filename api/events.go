package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/messaging"
	"go.uber.org/zap"
)

type EventCreateRequestBody struct {
	Title       string
	Description string
	When        string
	Where       string
	Need        string
}

type Event struct {
	ID      uint
	When    *time.Time
	Where   string
	Title   string
	Members []db.EventMember
	Need    int
}
type EventsResponse struct {
	Channels []*EventsResponseChannel
}

type EventsResponseChannel struct {
	*db.EventChannel
	Events []Event `json:"events"`
}

func init() {
	Router.Path("/api/v1/events").Methods("GET").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			eventsResponse := map[string]*EventsResponseChannel{}

			channels, err := DB.EventChannels()
			if err != nil {
				Logger.Error("could not get channels", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			for i := range channels {
				erc := &EventsResponseChannel{&channels[i], []Event{}}
				eventsResponse[erc.ChannelID] = erc
			}

			events, err := DB.Events()
			if err != nil {
				Logger.Error("could not get events", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// add the events to the correct eventsResponse
			for _, e := range events {
				if er, ok := eventsResponse[e.EventChannel.ChannelID]; ok {
					event := Event{
						ID:      e.ID,
						When:    e.When,
						Where:   e.Where,
						Title:   e.Title,
						Members: e.Members,
						Need:    e.Need,
					}
					er.Events = append(er.Events, event)
				}
			}
			json.NewEncoder(w).Encode(eventsResponse)
		},
	))

	// Create an event, needs when (time), where(channel id), title, and member from request
	Router.Path("/api/v1/events/create").Methods("POST").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			decoder := json.NewDecoder(r.Body)
			var data EventCreateRequestBody

			if err := decoder.Decode(&data); err != nil {
				Logger.Error("Unable to decode body", zap.Error(err))
			}
			// convert time
			timestamp, err := strconv.Atoi(data.When)
			if err != nil {
				Logger.Error("bad timestamp", zap.String("when", data.When))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// get the member
			member, err := DB.MemberByID(getMemberID(r))

			if err != nil {
				Logger.Error("couldn't find a member", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// get channel in DB
			eventChannel, err := DB.EventChannelByChannelID(data.Where)
			if err != nil {
				Logger.Error("Invalid event channel", zap.String("channel_id", data.Where), zap.Error(err))
			}

			// build the event
			event := DB.NewEvent()
			event.EventChannel = *eventChannel
			event.Title = data.Title
			event.Description = data.Description
			event.Need, _ = strconv.Atoi(data.Need)
			event.Members = []db.EventMember{
				{Member: *member, Type: db.EventMemberTypeHost}, // creator is automatically the host
			}
			if t := time.Unix(int64(timestamp), 0); true {
				event.When = &t
			}

			// save the event
			if err := event.Save(); err != nil {
				Logger.Error("could not save the event", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// send a message about the new event
			go messaging.SendNewEventMessage(event)

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

			// sort the channel names
			sort.Slice(eventChannels, func(i, j int) bool {
				iCh := fmt.Sprintf("%s:%s", eventChannels[i].ChannelCategoryName, eventChannels[i].ChannelName)
				jCh := fmt.Sprintf("%s:%s", eventChannels[j].ChannelCategoryName, eventChannels[j].ChannelName)
				return iCh < jCh
			})

			json.NewEncoder(w).Encode(eventChannels)
		},
	))
}
