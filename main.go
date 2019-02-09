//go:generate fileb0x ./b0x.json
package main

import (
	"fmt"
	"os"

	"io/ioutil"

	"github.com/FederationOfFathers/dashboard/api"
	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/config"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/environment"
	"github.com/FederationOfFathers/dashboard/events"
	"github.com/FederationOfFathers/dashboard/messaging"
	"github.com/FederationOfFathers/dashboard/metrics"
	"github.com/FederationOfFathers/dashboard/store"
	"github.com/FederationOfFathers/dashboard/streams"
	"github.com/apokalyptik/cfg"
	"github.com/bearcherian/rollzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

var twitchClientID = ""
var slackAPIKey = "xox...."
var slackMessagingKey = ""
var discordCfg = bot.DiscordCfg{}
var rollbarCfg = metrics.RollbarConfig{}
var logger *zap.Logger
var devPort = 0
var DB *db.DB
var mysqlURI string
var streamChannel = "-fof-dashboard"
var mindStreams bool

func init() {

	if environment.IsProd {
		logger, _ = zap.NewProduction()
	} else {
		logger, _ = zap.NewDevelopment()
	}
	logger = logger.Named("main")

	// ROLLBAR
	rollbarCore := rollzap.NewRollbarCore(zapcore.WarnLevel)
	if err := unmarshalConfig("cfg-rollbar.yml", &rollbarCfg); err != nil {
		logger.Error("Unable to unmarshal rollbar config", zap.Error(err))
	} else if rollbarCfg.Token != "" {
		rollbarCfg.Init()
		logger.Info("Rollbar initialized")
		logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, rollbarCore)
		}))
	}

	bot.Logger = logger.Named("bot")
	api.Logger = logger.Named("api")
	config.Logger = logger.Named("config")
	events.Logger = logger.Named("events")
	streams.Logger = logger.Named("streams")
	db.Logger = logger.Named("db")
	bridge.Logger = logger.Named("bridge")
	messaging.Logger = logger.Named("messaging")

	scfg := cfg.New("cfg-slack")
	scfg.StringVar(&slackAPIKey, "apiKey", slackAPIKey, "Slack API Key (env: SLACK_APIKEY)")
	scfg.StringVar(&slackMessagingKey, "messagingKey", slackMessagingKey, "Slack Messaging API Key (env: SLACK_MESSAGINGAPIKEY)")
	scfg.StringVar(&bot.CdnPrefix, "cdnPrefix", bot.CdnPrefix, "http url base from which to store saved uploads")
	scfg.StringVar(&bot.CdnPath, "cdnPath", bot.CdnPath, "Filesystem path to store uploads in")
	scfg.BoolVar(&bot.StartupNotice, "startupNotice", bot.StartupNotice, "send a start-up notice to slack")
	scfg.StringVar(&streamChannel, "streamChannel", streamChannel, "where to send streaming notices")
	scfg.BoolVar(&mindStreams, "mindStreams", mindStreams, "should we mind streaming?")

	err := unmarshalConfig("cfg-discord.yml", &discordCfg)
	if err != nil {
		logger.Error("Unable to load discord config", zap.Error(err))
	}

	acfg := cfg.New("cfg-api")
	acfg.StringVar(&api.ListenOn, "listen", api.ListenOn, "API bind address (env: API_LISTEN)")
	acfg.StringVar(&api.AuthSecret, "secret", api.AuthSecret, "Authentication secret for use in generating login tokens")
	acfg.StringVar(&api.JWTSecret, "hmac", api.JWTSecret, "Authentication secret used for JWT tokens")

	ecfg := cfg.New("cfg-events")
	ecfg.StringVar(&events.SaveFile, "savefile", events.SaveFile, "path to the file in which events should be persisted")
	ecfg.DurationVar(&events.SaveInterval, "saveinterval", events.SaveInterval, "how often to check and see if we need to save data")
	ecfg.StringVar(&events.OldEventLinkHMAC, "hmackey", events.OldEventLinkHMAC, "hmac key for generating team tool login links")

	dcfg := cfg.New("cfg-db")
	dcfg.StringVar(&mysqlURI, "mysql", mysqlURI, "MySQL Connection URI")
	dcfg.StringVar(&store.DBPath, "path", store.DBPath, "Path to the database file")

	tcfg := cfg.New("cfg-twitch")
	tcfg.StringVar(&twitchClientID, "clientID", "", "Twitch OAuth key")
}

func main() {
	cfg.Parse()

	store.Mind()

	DB = db.New("mysql", mysqlURI)
	streams.DB = DB
	api.DB = DB
	bot.DB = DB
	bot.AuthTokenGenerator = api.GenerateValidAuthTokens
	if environment.IsProd {
		bot.LoginLink = fmt.Sprintf("http://dashboard.fofgaming.com/")
	} else {
		bot.LoginLink = fmt.Sprintf("http://fofgaming.com%s/", api.ListenOn)
	}

	// start a separate message Slack connection if there is a separate messaging key
	if slackMessagingKey != "" {
		bot.MessagingKey = slackMessagingKey
	} else {
		bot.MessagingKey = slackAPIKey
	}
	err := bot.SlackConnect(slackAPIKey)
	if err != nil {
		logger.Fatal("Unable to contact the slack API", zap.Error(err))
	}
	slackApi := bot.NewSlackAPI(bot.MessagingKey, streamChannel)
	slackApi.Connect()
	defer slackApi.Shutdown()
	messaging.AddMsgAPI(slackApi)

	bridge.SlackCoreDataUpdated = bot.SlackCoreDataUpdated
	bridge.OldEventToolLink = events.OldEventToolLink
	bridge.OldEventToolAuthorization = events.OldEventToolAuthorization

	// start discord bot
	if discordCfg.Token != "" {
		logger.Info("Starting discord")
		discordApi := bot.NewDiscordAPI(discordCfg)
		discordApi.Connect()
		if discordCfg.RoleCfg.ChannelId != "" {
			discordApi.StartRoleHandlers()
		}
		defer discordApi.Shutdown()
		messaging.AddMsgAPI(discordApi)
	}

	streams.Init(streamChannel)
	if mindStreams {
		logger.Info("Minding streams", zap.String("channel", streamChannel))
		streams.MustTwitch(twitchClientID)
		streams.Mind()
	} else {
		streams.MindList()
		logger.Info("Not minding streams")
	}

	events.Start()
	api.Run()
}

// unmarshal a config YML file into an interface
func unmarshalConfig(fileName string, cfgObject interface{}) error {
	// exit quietly if no file. assume we are not configuring that portion
	if _, err := os.Stat(fileName); err != nil {
		logger.Info("File does not exist", zap.String("file", fileName))
		return nil
	}

	// read file data
	fileData, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	// unmarshal into interface object
	err2 := yaml.Unmarshal(fileData, cfgObject)
	if err2 != nil {
		return err2
	}

	return nil
}
