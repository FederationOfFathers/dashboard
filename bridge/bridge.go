package bridge

import "github.com/nlopes/slack"

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
	Events EventData
}{}

var SendMessage func(string, string)
var PostMessage func(string, string, slack.PostMessageParameters) error
