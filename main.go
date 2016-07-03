package main

import (
	"github.com/FederationOfFathers/dashboard/api"
	"github.com/FederationOfFathers/dashboard/events"
	"github.com/FederationOfFathers/dashboard/slack"
	"github.com/apokalyptik/cfg"
	"github.com/uber-go/zap"
)

var slackApiKey = "xox...."
var logger = zap.NewJSON()

func init() {
	scfg := cfg.New("cfg-slack")
	scfg.StringVar(&slackApiKey, "apiKey", slackApiKey, "Slack API Key (env: SLACK_APIKEY)")
	scfg.StringVar(&bot.CdnPrefix, "cdnPrefix", bot.CdnPrefix, "http url base from which to store saved uploads")
	scfg.StringVar(&bot.CdnPath, "cdnPath", bot.CdnPath, "Filesystem path to store uploads in")

	acfg := cfg.New("cfg-api")
	acfg.StringVar(&api.ListenOn, "listen", api.ListenOn, "API bind address (env: API_LISTEN)")

	ecfg := cfg.New("cfg-events")
	ecfg.StringVar(&events.SaveFile, "savefile", events.SaveFile, "path to the file in which events should be persisted")
	ecfg.DurationVar(&events.SaveInterval, "saveinterval", events.SaveInterval, "how often to check and see if we need to save data")

}

func main() {
	cfg.Parse()
	data, err := bot.SlackConnect(slackApiKey)
	events.Start(data)
	if err != nil {
		logger.Fatal("Unable to contact the slack API", zap.Error(err))
	}
	api.Run(data, events.Data)
}
