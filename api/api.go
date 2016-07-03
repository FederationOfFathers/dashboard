package api

import (
	"github.com/FederationOfFathers/dashboard/events"
	"github.com/FederationOfFathers/dashboard/slack"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/phyber/negroni-gzip/gzip"
	"github.com/rs/cors"
	"github.com/uber-go/zap"
)

var ListenOn = ":8866"
var router = mux.NewRouter()
var logger = zap.NewJSON().With(zap.String("module", "api"))

var slackData *bot.SlackData
var eventData *events.Events

func Run(slData *bot.SlackData, eData *events.Events) {
	slackData = slData
	eventData = eData
	n := negroni.New()
	n.Use(&httpLogger{})
	n.Use(cors.New(cors.Options{AllowedOrigins: []string{"*"}}))
	n.Use(negroni.NewRecovery())
	n.Use(gzip.Gzip(gzip.DefaultCompression))
	n.UseHandler(router)
	n.Run(ListenOn)
}
