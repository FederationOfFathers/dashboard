package bot

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

var isDev = os.Getenv("DEV_LOGGING") != ""

type MessageHandler func(*slack.MessageEvent) bool

var ChannelMessageHandlers = []MessageHandler{
	handleJoinPartEvents,
	handleTimeoutMessages,
	handleChannelUpload,
	handleFortune,
	handleSaySomething,
	handleDice,
	handleTimeout,
}
var GroupMessageHandlers = []MessageHandler{
	handleJoinPartEvents,
	handleChannelUpload,
	handleFortune,
	handleSaySomething,
	handleDice,
	handleTimeout,
}
var DirectMessageHandlers = []MessageHandler{
	handleLoginCode,
	handleLogin,
	handleDMUpload,
	handleFortune,
	handleSaySomething,
	handleDevLogin,
	handleDice,
	handleTimeout,
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

func handleJoinPartEvents(m *slack.MessageEvent) bool {
	switch m.SubType {
	case "channel_leave", "channel_join", "group_leave", "group_join":
		UpdateRequest <- struct{}{}
		return true
	}
	return false
}

func handleFortune(m *slack.MessageEvent) bool {
	if atBotPrefixed(m.Msg.Text) {
		parsed := strings.SplitN(m.Msg.Text, " ", 3)
		if parsed[1] == "fortune" {
			if out, err := exec.Command("fortune").Output(); err != nil {
				Logger.Error("error running the fortune command", zap.Error(err))
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

var diceRx = regexp.MustCompile("^roll ([0-9]+)?(d)([0-9]+)(?:([+-])([0-9]+))?\\s*$")

func handleDice(m *slack.MessageEvent) bool {
	if isDev {
		return false
	}
	if roll := diceRx.FindAllStringSubmatch(m.Msg.Text, -1); roll != nil {
		var err error
		var mult = 1
		var max = 1
		var plus = 0
		var minus = 0
		var total = 0
		var dice []int

		if roll[0][1] != "" {
			mult, err = strconv.Atoi(roll[0][1])
			if err != nil {
				return false
			}
		}
		max, err = strconv.Atoi(roll[0][3])
		if err != nil {
			return false
		}
		if roll[0][4] != "" && roll[0][5] != "" {
			if roll[0][4] == "+" {
				plus, err = strconv.Atoi(roll[0][5])
				if err != nil {
					return false
				}
			} else {
				minus, err = strconv.Atoi(roll[0][5])
				if err != nil {
					return false
				}
			}
		}
		var out string
		var strdice = []string{}
		for i := 0; i < mult; i++ {
			die := rand.Intn(max-1) + 1
			dice = append(dice, die)
			strdice = append(strdice, fmt.Sprintf("%d", die))
			total = total + die
		}
		if plus > 0 {
			total = total + plus
			out = fmt.Sprintf("+%d", plus)
		}
		if minus > 0 {
			total = total - minus
			out = fmt.Sprintf("-%d", minus)
		}
		out = fmt.Sprintf("*%d* (%s%s)", total, strings.Join(strdice, ","), out)
		rtm.SendMessage(&slack.OutgoingMessage{
			ID:      int(time.Now().UnixNano()),
			Channel: m.Channel,
			Text:    fmt.Sprintf("rolled %s", out),
			Type:    "message",
		})
	}
	return false
}
