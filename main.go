//go:generate fileb0x ./b0x.json
package main

import (
	"context"
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
	"github.com/honeycombio/beeline-go"
	rollbar "github.com/rollbar/rollbar-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"gopkg.in/yaml.v2"
)

var twitchClientID = ""
var twitchClientSecret = ""
var youtubeAPIKey = ""
var discordCfg = bot.DiscordCfg{}
var rollbarCfg = metrics.RollbarConfig{}
var logger *zap.Logger
var devPort = 0
var DB *db.DB
var mysqlURI string
var streamChannel = "-fof-dashboard"
var mindStreams bool
var honeycombToken string
var honeycombDataset string = "unknown"

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
	tcfg.StringVar(&twitchClientID, "clientID", "", "Twitch Client ID")
	tcfg.StringVar(&twitchClientSecret, "clientSecret", "", "Twitch Client Secret")

	ytcfg := cfg.New("cfg-youtube")
	ytcfg.StringVar(&youtubeAPIKey, "apiKey", "", "YouTube API Key")

	hcfg := cfg.New("cfg-honeycomb")
	hcfg.StringVar(&honeycombToken, "token", honeycombToken, "Token for Honeycomb project reporting")
	hcfg.StringVar(&honeycombDataset, "dataset", honeycombDataset, "Dataset for Honeycomb project reporting")
}

func main() {

	// panic recovery/reporting/logging
	defer func() {
		if err := recover(); err != nil {
			logger.With(zap.Any("error", err)).Error("APPLICATION PANIC")
		}
	}()
	cfg.Parse()

	if honeycombToken != "" {
		logger.Info("setting up honeycomb")
		beeline.Init(beeline.Config{
			WriteKey:    honeycombToken,
			Dataset:     honeycombDataset,
			ServiceName: "dashboard",
		})
	}

	store.Mind()

	DB = db.New("mysql", mysqlURI)
	streams.DB = DB
	api.DB = DB
	bot.DB = DB
	events.DB = DB

	bridge.DiscordCoreDataUpdated = bot.DiscordCoreDataUpdated
	bridge.OldEventToolLink = events.OldEventToolLink
	bridge.OldEventToolAuthorization = events.OldEventToolAuthorization

	var yt *youtube.Service
	if youtubeAPIKey != "" {
		y, err := youtube.NewService(context.Background(), option.WithAPIKey(youtubeAPIKey))
		if err != nil {
			logger.Error("YouTube service creation failed", zap.Error(err))
		}
		yt = y
		logger.Info("YouTube service created")
	}

	streams.YouTube = yt

	// start discord bot
	if discordCfg.Token != "" {
		if err := startDiscord(discordCfg, yt); err != nil {
			logger.Error(fmt.Sprintf("Discord failed to start: %s", err))
		}
	}

	if mindStreams {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("Unable to mind streams", zap.String("reocvered", fmt.Sprintf("%v", r)))
				}
			}()
			logger.Info("Minding streams", zap.String("channel", streamChannel))
			if err := streams.Twitch(twitchClientID, twitchClientSecret); err != nil {
				logger.Error("unable to init Twitch client", zap.Error(err))
			}
			streams.Mind()
		}()

	} else {
		logger.Info("Not minding streams")
	}

	events.Start() //TODO still needed? old events?
	events.MindEvents()

	rollbar.Info("starting up")
	api.Run()

}

func startDiscord(discordCfg bot.DiscordCfg, yt *youtube.Service) error {
	logger.Info("Starting discord")
	discordApi, err := bot.StartDiscord(discordCfg, yt)
	if err != nil {
		return err
	}
	discordApi.MindGuild()
	defer discordApi.Shutdown()

	messaging.AddMsgAPI(discordApi)

	return nil
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
