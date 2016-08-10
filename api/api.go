package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/events"
	"github.com/FederationOfFathers/dashboard/slack"
	"github.com/gorilla/mux"
	"github.com/uber-go/zap"
)

var ListenOn = ":8866"
var Router = mux.NewRouter()
var logger = zap.NewJSON().With(zap.String("module", "api"))

var slackData *bot.SlackData
var eventData *events.Events

var URLHostName = ""
var URLPrefix = ""
var URLScheme = "https"

func myURL() string {
	return fmt.Sprintf("%s://%s:%s%s", URLScheme, URLHostName, ListenOn, URLPrefix)
}

func Run(slData *bot.SlackData, eData *events.Events) {
	visDB = store.DB.Groups().NewNestedStore([]byte("visibility-v1"))
	if URLHostName == "" {
		URLHostName, _ = os.Hostname()
	}
	slackData = slData
	eventData = eData
	logger.Fatal("error starting API http server", zap.String("listenOn", ListenOn), zap.Error(http.ListenAndServe(ListenOn, Router)))
}
