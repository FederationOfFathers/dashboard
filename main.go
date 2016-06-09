package main

import (
	"github.com/FederationOfFathers/dashboard-api"
	"github.com/FederationOfFathers/dashboard-slack"
	"github.com/apokalyptik/cfg"
	"github.com/uber-go/zap"
)

var slackApiKey = "xox...."
var logger = zap.NewJSON()

func init() {
	scfg := cfg.New("slack")
	scfg.StringVar(&slackApiKey, "apiKey", slackApiKey, "Slack API Key (env: SLACK_APIKEY)")
}

func main() {
	cfg.Parse()
	logger.SetLevel(zap.Info)
	data, err := bot.SlackConnect(slackApiKey)
	if err != nil {
		logger.Fatal("Unable to contact the slack API", zap.Err(err))
	}
	api.Run(data)
}
