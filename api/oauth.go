package api

import (
	"net/http"
	"github.com/gorilla/mux"
	"strings"
	"errors"
)

func init() {
	Router.Path("/api/v0/oauth/{service}").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			service := mux.Vars(r)["service"]
			switch service {
			case "youtube":
				handleYoutube(w, r)
			}
		},
	))
}
func handleYoutube(w http.ResponseWriter, r *http.Request) {
	token, err := parseYoutubeToken(r.URL.Fragment)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		InsertMemberMetaBySlackId(getSlackUserID(r), "ytAccessToken", token)
	}
}
func parseYoutubeToken(urlFragment string) (string, error)  {
	var err error
	splitFragment := strings.SplitN(urlFragment, "&", -1)
	tokenSlice := strings.SplitN(splitFragment[1], "=", -1)
	if len(tokenSlice) < 2 {
		err = errors.New("No token specified")
	}
	return tokenSlice[1], err
}
