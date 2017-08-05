package api

import (
	"bytes"
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

var BattleNetKey string
var BattleNetSecret string

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
			case "battlenet":
				handleBattleNet(w, r)
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

// Blizzard/Battle.net's OAuth flow is done primarily server side.
// They don't seem to have a client side flow that encompasses the entire flow
// an authorization code is passed in, we then use that to get a users access token, and then get their battle tag using that access token
func handleBattleNet(w http.ResponseWriter, r *http.Request) {
	auth_code, err := parseToken(r.URL.Fragment, "code")
	if err != nil || auth_code == "" {
		w.Write([]byte("Invalid code provided"))
		w.WriteHeader(http.StatusBadRequest)
	} else {

		// get access token with provided code
		access_token, tokenError := getBNetAccessToken(auth_code, r.URL.Query().Get("redirect_uri"))
		if tokenError != nil {
			logger.Error(fmt.Sprintf("Unable to get access token: %v", tokenError))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// get BattleTag with access token
		battleTag, btError := getBattleTag(access_token)
		if btError != nil {
			logger.Error(fmt.Sprintf("Unable to get BattleTag. %v", btError))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// add user name to db
		slackUserId := getSlackUserID(r)
		userError := InsertMemberMetaBySlackId(slackUserId, "bnet_username", battleTag)
		if userError != nil {
			logger.Error(fmt.Sprintf("Unable to update BattleTag data. %v", userError))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func getBattleTag(access_token string) (string, error) {
	userUrl := fmt.Sprintf("https://us.api.battle.net/account/user?access_token=%s", access_token)

	userResponse := BNetUserResponse{}
	userError := getJsonResponse(userUrl, &userResponse)
	if userError != nil {
		return "", fmt.Errorf("Unable to get user account. %v", userError)
	}

	return userResponse.BattleTag, nil
}

func getBNetAccessToken(auth_code string, redirectUri string) (string, error) {
	tokenUrl := fmt.Sprintf("https://us.battle.net/oauth/token?redirect_uri=%s&grant_type=%s&code=%s", redirectUri, "client_credentials", auth_code)
	tokenResponse := BNetAccessTokenResponse{}

	tokenError := getAuthenticatedJsonResponse(tokenUrl, &tokenResponse, BattleNetKey, BattleNetSecret)
	if tokenError != nil {
		return "", fmt.Errorf("Unable to get token: %v", tokenError)
	}

	if tokenResponse.Error != "" {
		return "", fmt.Errorf("API returned and error response - %s:%s", tokenResponse.Error, tokenResponse.ErrorDescription)
	} else {
		return tokenResponse.AccessToken, nil
	}

}

func getJsonResponse(url string, target interface{}) error {
	return getAuthenticatedJsonResponse(url, target, "", "")
}

func getAuthenticatedJsonResponse(url string, target interface{}, user string, password string) error {
	client := &http.Client{
		Timeout: time.Duration(30 * time.Second),
	}
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBufferString(""))
	if err != nil {
		return fmt.Errorf("Unable to create access token request: %v", err)
	}

	req.Header.Add("Accept", "application/json")
	if user != "" && password != "" {
		req.SetBasicAuth(user, password)
	}

	resp, clientRequestError := client.Do(req)
	if clientRequestError != nil {
		return fmt.Errorf("Unable to complete request: %v", clientRequestError)
	}
	defer resp.Body.Close()

	jsonErr := json.NewDecoder(resp.Body).Decode(&target)
	if jsonErr != nil {
		return fmt.Errorf("Unable to parse json: %v", jsonErr)
	}

	return nil
}

func isValidState(slackId string, service string, encodedState string) bool {
	stateKey := fmt.Sprintf(SERVICE_STATE_FORMAT, service)
	metaEntry := getMemberMetaEntryBySlackID(slackId, stateKey)
	encodedDBState := base64.StdEncoding.EncodeToString(metaEntry.MetaJSON)
	thirtyMinutesAgo := time.Now().Add(-30 * time.Minute)
	return metaEntry.UpdatedAt.After(thirtyMinutesAgo) && encodedDBState == encodedState

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

type BNetAccessTokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        string `json:"expired_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type BNetUserResponse struct {
	Id        int32  `json:"id"`
	BattleTag string `json:"battletag"`
}
