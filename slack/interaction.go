package bot

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
)

type MessageHandler func(*slack.MessageEvent) bool
type AuthTokenGeneratorFunc func(string) []string

var AuthTokenGenerator = func(s string) []string {
	return nil
}

var ChannelMessageHandlers = []MessageHandler{
	handleChannelUpload,
	handleFortune,
	handleSaySomething,
}
var GroupMessageHandlers = []MessageHandler{
	handleChannelUpload,
	handleFortune,
	handleSaySomething,
}
var DirectMessageHandlers = []MessageHandler{
	handleLogin,
	handleDMUpload,
	handleFortune,
	handleSaySomething,
}

func handleChannelMessage(m *slack.MessageEvent) bool {
	for _, handler := range ChannelMessageHandlers {
		if handler(m) {
			return true
		}
	}
	return false
}

func handleGroupMessage(m *slack.MessageEvent) bool {
	for _, handler := range GroupMessageHandlers {
		if handler(m) {
			return true
		}
	}
	return true
}

func handleDirectMessage(m *slack.MessageEvent) bool {
	for _, handler := range DirectMessageHandlers {
		if handler(m) {
			return true
		}
	}
	return true
}

func atBotPrefixed(message string) bool {
	if strings.HasPrefix(message, fmt.Sprintf("%s: ", connection.User.Name)) {
		return true
	}
	if strings.HasPrefix(message, fmt.Sprintf("%s ", connection.User.Name)) {
		return true
	}
	if strings.HasPrefix(message, fmt.Sprintf("<@%s>", connection.User.ID)) {
		return true
	}
	return false
}

func handleSaySomething(m *slack.MessageEvent) bool {
	if !atBotPrefixed(m.Msg.Text) {
		return false
	}
	parsed := strings.SplitN(m.Msg.Text, " ", 4)
	if len(parsed) < 4 {
		return false
	}
	if parsed[1] == "say" {
		admin, err := IsUserIDAdmin(m.User)
		if !admin {
			return false
		}
		if err != nil {
			rtm.PostMessage(m.Channel, err.Error(), slack.PostMessageParameters{})
			return false
		}
		var to string
		if parsed[2][:1] == "<" {
			switch parsed[2][1:2] {
			case "@":
				if _, _, channel, err := rtm.OpenIMChannel(strings.Trim(parsed[2], "@<>")); err != nil {
					outgoingMessage := &slack.OutgoingMessage{
						ID:      int(time.Now().UnixNano()),
						Channel: m.Msg.Channel,
						Text:    fmt.Sprintf("Error opening IM: %s", err.Error()),
						Type:    "message",
					}
					rtm.SendMessage(outgoingMessage)
				} else {
					to = channel
				}
			case "#":
				to = strings.Trim(parsed[2], "#<>")
			}
		}
		if to == "" {
			guess := strings.Trim(parsed[2], "<>@:#-=+!@#$%^&*(){}[]\\|<>?,./")
			for _, g := range data.GetGroups() {
				if g.Name == guess {
					to = g.ID
					break
				}
			}
			for _, u := range data.GetUsers() {
				if strings.ToLower(u.Name) == strings.ToLower(guess) {
					if _, _, channel, err := rtm.OpenIMChannel(u.ID); err == nil {
						to = channel
						break
					}
				}
			}
			for _, c := range data.GetChannels() {
				if c.Name == guess {
					to = c.ID
				}
			}
		}
		if to == "" {
			rtm.SendMessage(&slack.OutgoingMessage{
				ID:      int(time.Now().UnixNano()),
				Channel: m.Msg.Channel,
				Text:    fmt.Sprintf("who's %s?", parsed[2]),
				Type:    "message",
			})
			return false
		}
		rtm.SendMessage(&slack.OutgoingMessage{
			ID:      int(time.Now().UnixNano()),
			Channel: to,
			Text:    parsed[3],
			Type:    "message",
		})
		return true
	}
	return false
}

func handleFortune(m *slack.MessageEvent) bool {
	if atBotPrefixed(m.Msg.Text) {
		parsed := strings.SplitN(m.Msg.Text, " ", 3)
		if parsed[1] == "fortune" {
			if out, err := exec.Command("fortune").Output(); err != nil {
				logger.Error("error running the fortune command", zap.Error(err))
			} else {
				rtm.SendMessage(&slack.OutgoingMessage{
					ID:      int(time.Now().UnixNano()),
					Channel: m.Channel,
					Text:    fmt.Sprintf("```%s```", string(out)),
					Type:    "message",
				})
			}
			return true
		}
	}
	return false
}

func handleLogin(m *slack.MessageEvent) bool {
	if m.Msg.Text != "login" {
		return false
	}

	if AuthTokenGenerator == nil {
		return false
	}

	rtm.SendMessage(&slack.OutgoingMessage{
		ID:      int(time.Now().UnixNano()),
		Channel: m.Channel,
		Text:    fmt.Sprintf("%s -- %s", m.Msg.User, AuthTokenGenerator(m.Msg.User)[0]),
		Type:    "message",
	})

	return true
}
