package bridge

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/nlopes/slack"
)

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
		log.Printf("error updating seen in %s: %s", time.Now().Sub(begin).String(), err.Error())
		return
	}
	err = json.NewDecoder(rsp.Body).Decode(&newSeen)
	if err != nil {
		log.Printf("error updating seen in %s: %s", time.Now().Sub(begin).String(), err.Error())
		return
	}
	if len(newSeen) > 100 {
		log.Printf("updated seen in %s", time.Now().Sub(begin).String())
		Data.Seen = newSeen
	} else {
		log.Printf("error updating seen in %s: seen is strangely empty...", time.Now().Sub(begin).String())
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
