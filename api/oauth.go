package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const SERVICE_STATE_FORMAT string = "%s_state"

func init() {
	Router.Path("/api/v0/oauth/{service}").Methods("POST", "OPTIONS").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			userId := getSlackUserID(r)
			service := mux.Vars(r)["service"]
			encodedState, _ := parseToken(r.URL.Fragment, "state")

			if !isValidState(userId, service, encodedState) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			switch service {
			case "youtube":
				handleYoutube(w, r)
			}
		},
	))

	Router.Path("/api/v0/oauth/{service}/state").Methods("GET", "OPTIONS").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			service := mux.Vars(r)["service"]
			userId := getSlackUserID(r)
			serviceStateMetaKey := fmt.Sprintf(SERVICE_STATE_FORMAT, service)

			//get state meta
			meta := getMemberMetaEntryBySlackID(userId, serviceStateMetaKey)
			state := string(meta.MetaJSON)
			encodedState := base64.StdEncoding.EncodeToString([]byte(state))

			if state == "" || !isValidState(userId, service, encodedState) {
				state = fmt.Sprintf("%s:%s:%s", userId, service, strconv.FormatInt(time.Now().Unix(), 10))
				encodedState = base64.StdEncoding.EncodeToString([]byte(state))
				InsertMemberMetaBySlackId(userId, serviceStateMetaKey, state)
			}

			logger.Info(fmt.Sprintf("STATE: %v", state))
			stateMap := make(map[string]string)
			stateMap["state"] = encodedState

			json.NewEncoder(w).Encode(stateMap)
		},
	))
}

func handleYoutube(w http.ResponseWriter, r *http.Request) {
	token, err := parseToken(r.URL.Fragment, "access_token")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		InsertMemberMetaBySlackId(getSlackUserID(r), "ytAccessToken", token)
	}
}
func isValidState(slackId string, service string, encodedState string) bool {
	stateKey := fmt.Sprintf(SERVICE_STATE_FORMAT, service)
	metaEntry := getMemberMetaEntryBySlackID(slackId, stateKey)
	encodedDBState := base64.StdEncoding.EncodeToString(metaEntry.MetaJSON)
	thirtyMinutesAgo := time.Now().Add(-30 * time.Minute)
	return  metaEntry.UpdatedAt.After(thirtyMinutesAgo) && encodedDBState == encodedState

}

func parseToken(urlFragment string, tokenName string) (string, error) {
	var err error
	params := strings.SplitN(urlFragment, "&", -1)
	for _, param := range params {
		kv := strings.SplitN(param, "=", -1)
		var key = kv[0]
		var val = kv[1]
		if key == tokenName {
			return val, err
		}
	}

	return "", err
}
