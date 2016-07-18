package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/FederationOfFathers/dashboard/events"
	"github.com/FederationOfFathers/dashboard/slack"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/rabeesh/negroni-nocache"
	"github.com/rs/cors"
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
	if URLHostName == "" {
		URLHostName, _ = os.Hostname()
	}
	slackData = slData
	eventData = eData
	n := negroni.New()
	n.Use(&httpLogger{})
	n.Use(cors.New(cors.Options{AllowedOrigins: []string{"*"}}))
	n.Use(negroni.NewRecovery())
	n.Use(nocache.New(true))
	//n.Use(gzip.Gzip(gzip.DefaultCompression))
	n.UseHandler(Router)
	logger.Fatal("error starting API http server", zap.String("listenOn", ListenOn), zap.Error(http.ListenAndServe(ListenOn, n)))
}
