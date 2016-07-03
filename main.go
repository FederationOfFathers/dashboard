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
var cdnPath = ""
var cdnPrefix = ""

func init() {
	scfg := cfg.New("slack")
	scfg.StringVar(&slackApiKey, "apiKey", slackApiKey, "Slack API Key (env: SLACK_APIKEY)")
	scfg.StringVar(&cdnPrefix, "cdnPrefix", cdnPrefix, "http url base from which to store saved uploads")
	scfg.StringVar(&cdnPath, "cdnPath", cdnPath, "Filesystem path to store uploads in")
}

func main() {
	cfg.Parse()
	bot.CdnPath = cdnPath
	bot.CdnPrefix = cdnPrefix
	data, err := bot.SlackConnect(slackApiKey)
	events.Start(data)
	if err != nil {
		logger.Fatal("Unable to contact the slack API", zap.Error(err))
	}
	api.Run(data, events.Data)
}
