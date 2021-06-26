package bridge

import (
	"sync"

	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

var Logger *zap.Logger

type EventData interface{}

var Data = &struct {
	Events EventData
}{}
var OldEventToolAuthorization func(string) string
var OldEventToolLink func(string) string
var DiscordCoreDataUpdated *sync.Cond
var SendMessage func(string, string)
var PostMessage func(string, string, slack.PostMessageParameters) error
