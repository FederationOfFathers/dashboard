package messaging

import (
	"reflect"

	"go.uber.org/zap"

	"github.com/FederationOfFathers/dashboard/db"
)

var msgApis []MsgAPI
var Logger *zap.Logger

type StreamMessage struct {
	Platform         string
	PlatformLogo     string
	PlatformColor    string
	PlatformColorInt int
	Username         string
	UserLogo         string
	URL              string
	Description      string
	Timestamp        string
	ThumbnailURL     string
}

// Add Messaging APIs that will be used to send messages
func AddMsgAPI(msgApi MsgAPI) {
	Logger.Info("Adding new message API", zap.String("type", reflect.TypeOf(msgApi).String()))
	msgApis = append(msgApis, msgApi)
}

type MsgAPI interface {
	PostStreamMessage(sm StreamMessage) error
	PostNewEventMessage(e *db.Event) error
	PostJoinEventMessage(e *db.Event, member string) error
	//PostMessageToChannel(channel string, message string)
}

func SendTwitchStreamMessage(sm StreamMessage) {
	postStreamMessageToAllApis(sm)
}

func SendNewEventMessage(e *db.Event) {
	for _, msgApi := range msgApis {
		err := msgApi.PostNewEventMessage(e)
		if err != nil {
			Logger.Error("unable to send event notice", zap.Any("event", e), zap.Error(err))
		}
	}
}

func SendJoinEventMessage(e *db.Event, member *db.Member) {
	for _, msgApi := range msgApis {
		err := msgApi.PostJoinEventMessage(e, member.Name)
		if err != nil {
			Logger.Error("unable to send event join message", zap.Any("event", e), zap.String("member", member.Discord), zap.Error(err))
		}
	}
}

func postStreamMessageToAllApis(sm StreamMessage) {
	Logger.Info("sending stream message", zap.String("username", sm.Username), zap.String("platform", sm.Platform))
	for _, msgApi := range msgApis {
		err := msgApi.PostStreamMessage(sm)
		if err != nil {
			Logger.With(zap.Any("message", sm)).Error("unable to send stream update", zap.Error(err))
		}
	}
}
