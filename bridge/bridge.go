package bridge

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

var Logger *zap.Logger

type SlackData interface {
	IsUsernameAdmin(string) (bool, error)
	IsUserIDAdmin(string) (bool, error)

	ChannelByName(string) (*slack.Channel, error)
	GetChannels() []slack.Channel
	UserChannels(string) []slack.Channel

	User(string) (*slack.User, error)
	GetUsers() []slack.User
	UserByName(string) (*slack.User, error)

	UserGroups(string) []slack.Group
	GetGroups() []slack.Group
	GroupByName(string) (*slack.Group, error)
}

type EventData interface{}

var Data = &struct {
	Slack  SlackData
	Seen   map[string]time.Time
	Events EventData
}{
	Seen: map[string]time.Time{},
}
var OldEventToolAuthorization func(string) string
var OldEventToolLink func(string) string
var SlackCoreDataUpdated *sync.Cond
var SendMessage func(string, string)
var PostMessage func(string, string, slack.PostMessageParameters) error

func updateSeen() {
	begin := time.Now()
	var newSeen = map[string]time.Time{}
	rsp, err := http.Get("http://fofgaming.com:8890/seen.json")
	defer rsp.Body.Close()
	if err != nil {
		Logger.Error("fetching", zap.Error(err))
		return
	}
	err = json.NewDecoder(rsp.Body).Decode(&newSeen)
	if err != nil {
		Logger.Error("decoding", zap.Error(err))
		return
	}
	if len(newSeen) > 100 {
		Logger.Debug("updated seen", zap.Duration("took", time.Now().Sub(begin)))
		Data.Seen = newSeen
	} else {
		Logger.Error("seen is empty!")
	}
}

func init() {
	go updateSeen()
	go func() {
		t := time.Tick(time.Second * 30)
		for {
			select {
			case <-t:
				updateSeen()
			}
		}
	}()
}
