package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/gorilla/mux"
	"github.com/uber-go/zap"
)

var ListenOn = ":8866"
var Router = mux.NewRouter()
var logger = zap.NewJSON().With(zap.String("module", "api"))

var slackData = bridge.Data.Slack
var eventData = bridge.Data.Events

var URLHostName = ""
var URLPrefix = ""
var URLScheme = "https"

func myURL() string {
	return fmt.Sprintf("%s://%s:%s%s", URLScheme, URLHostName, ListenOn, URLPrefix)
}

func Run() {
	if URLHostName == "" {
		URLHostName, _ = os.Hostname()
	}
	logger.Fatal("error starting API http server", zap.String("listenOn", ListenOn), zap.Error(http.ListenAndServe(ListenOn, Router)))
}
