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
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type EventCreateRequestBody struct {
	Title       string
	Description string
	When        string
	Where       string
	Need        int
}

type EventJoinRequestBody struct {
	Type int `json:"type"`
}

type EventLeaveRequestBody struct {
	// Member is the id of the event_member (EventMember.ID), not the member id (Member.ID). One member can fill multiple slots in an event.
	Member uint `json:"member"`
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
	// get events grouped by channel
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
				eventsResponse[erc.ID] = erc
			}

			events, err := DB.Events()
			if err != nil {
				Logger.Error("could not get events", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// add the events to the correct eventsResponse
			for _, e := range events {
				if er, ok := eventsResponse[e.EventChannelID]; ok {
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

			id := getMemberID(r)
			mid, err := strconv.Atoi(id)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			// get the member
			member, err := DB.MemberByID(mid)

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
			event.EventChannelID = data.Where
			event.Title = data.Title
			event.Description = data.Description
			event.Need = data.Need
			event.Members = []db.EventMember{
				{MemberID: member.ID, Type: db.EventMemberTypeHost}, // creator is automatically the host
			}
			if t := time.Unix(int64(timestamp), 0); true {
				event.When = &t
			}

			// save the event
			if err := event.Save(); err != nil {
				Logger.Error("could not save the event", zap.Any("event", event), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// send a message about the new event
			go messaging.SendNewEventMessage(event)

			json.NewEncoder(w).Encode(event)
		},
	))

	// Join an event
	Router.Path("/api/v1/events/{eventID}/join").Methods("POST").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set("Content-Type", "application/json")

			decoder := json.NewDecoder(r.Body)
			vars := mux.Vars(r)
			var data EventJoinRequestBody

			if err := decoder.Decode(&data); err != nil {
				Logger.Error("Unable to decode body", zap.Error(err))
			}

			id := getMemberID(r)
			mid, err := strconv.Atoi(id)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			// member
			member, err := DB.MemberByID(mid)
			if err != nil {
				Logger.Error("could not get a valid member", zap.Error(err))
			}

			//event
			eventID, err := strconv.Atoi(vars["eventID"])
			if err != nil {
				Logger.Error("invalid eventID", zap.String("eventID", vars["eventID"]), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			event, err := DB.EventByID(eventID)
			if err != nil {
				Logger.Error("unable to find event", zap.Int("eventID", eventID), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			event.Members = append(event.Members, db.EventMember{MemberID: member.ID, Type: data.Type})
			if err := event.Save(); err != nil {
				Logger.Error("unable to save event", zap.Any("event", event), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}

			w.WriteHeader(http.StatusOK)

		},
	))

	// Leave an event
	Router.Path("/api/v1/events/leave").Methods("POST").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set("Content-Type", "application/json")

			decoder := json.NewDecoder(r.Body)
			//vars := mux.Vars(r)
			var data EventLeaveRequestBody

			if err := decoder.Decode(&data); err != nil {
				Logger.Error("Unable to decode body", zap.Error(err))
			}

			id := getMemberID(r)
			mid, err := strconv.Atoi(id)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			// logged in member
			member, err := DB.MemberByID(mid)
			if err != nil {
				Logger.Error("could not get a valid member", zap.Error(err))
				w.WriteHeader(http.StatusForbidden)
				return
			}

			eMember, err := DB.EventMemberByID(data.Member)
			if err != nil {
				Logger.Error("unable to delete event member", zap.Uint("member id", data.Member), zap.Error(err))
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if member.ID != eMember.MemberID {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			DB.DeleteEventMemberByID(data.Member)

			w.WriteHeader(http.StatusOK)

		},
	))

	// Delete event
	Router.Path("/api/v1/events/{eventID}").Methods("DELETE").Handler(authenticated(
		func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)

			id := getMemberID(r)
			mid, err := strconv.Atoi(id)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			// get member
			member, err := DB.MemberByID(mid)
			if err != nil {
				Logger.Error("invalid member", zap.String("memberid", id))
				w.WriteHeader(http.StatusForbidden)
				return
			}

			//get event
			eventID, err := strconv.Atoi(vars["eventID"])
			if err != nil {
				Logger.Error("invalid eventID", zap.String("eventID", vars["eventID"]), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			event, err := DB.EventByID(eventID)
			if err != nil {
				Logger.Error("unable to find", zap.Int("eventID", eventID), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			members, err := DB.EventMembers(event)
			if err != nil {
				Logger.Error("unable to find event members", zap.Any("event", event), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			isOwner := false
			for _, eM := range members {
				if eM.MemberID == member.ID && eM.Type == db.EventMemberTypeHost {
					isOwner = true
					break
				}
			}

			if isOwner {
				Logger.Info("Deleting event", zap.Any("event", event))
				DB.DeleteEvent(*event)
			} else {
				Logger.Debug("bad delete request from user", zap.Int("id", member.ID), zap.Any("event", event))
				w.WriteHeader(http.StatusForbidden)
				return
			}
		},
	))

	// get the channels
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
