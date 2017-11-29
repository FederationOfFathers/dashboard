package bot

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

type Timeouts struct {
	sync.Mutex
	data map[string]time.Time
}

var timeouts = Timeouts{
	Mutex: sync.Mutex{},
	data:  map[string]time.Time{},
}

func handleTimeout(m *slack.MessageEvent) bool {
	if !atBotPrefixed(m.Msg.Text) {
		return false
	}
	parsed := strings.SplitN(m.Msg.Text, " ", 4)
	if len(parsed) < 4 {
		return false
	}
	if parsed[1] == "timeout" {
		if admin, _ := IsUserIDAdmin(m.User); !admin {
			rtm.SendMessage(&slack.OutgoingMessage{
				ID:      int(time.Now().UnixNano()),
				Channel: m.Channel,
				Text:    fmt.Sprintf("Snitches get stitches... 5 minute timeout for you..."),
				Type:    "message",
			})
			Logger.Warn("user not admin... timeout")
			timeouts.Lock()
			timeouts.data[m.User] = time.Now().Add(5 * time.Minute)
			timeouts.Unlock()
			return false
		}
		guess := strings.Trim(parsed[2], "<>@:#-=+!@#$%^&*(){}[]\\|<>?,./")
		if d, err := time.ParseDuration(parsed[3]); err != nil {
			Logger.Error(
				"error parsing duration",
				zap.String("string", guess),
				zap.Error(err))
		} else {
			Logger.Warn("Adding timeout", zap.String("user", guess), zap.Duration("duration", d))
			timeouts.Lock()
			timeouts.data[guess] = time.Now().Add(d)
			timeouts.Unlock()
		}
	}
	return false
}

func handleTimeoutMessages(m *slack.MessageEvent) bool {
	if _, ok := timeouts.data[m.Msg.User]; !ok {
		return false
	}
	_, _, err := rtm.DeleteMessage(m.Msg.Channel, m.Msg.Timestamp)
	if err != nil {
		Logger.Error(
			"Error deleting timeout message",
			zap.String("username", m.Username),
			zap.String("filename", m.Msg.Text),
			zap.Error(err))
	}
	return true
}

func init() {
	go func() {
		t := time.Tick(time.Second)
		for {
			select {
			case <-t:
				now := time.Now()
				timeouts.Lock()
				for k, v := range timeouts.data {
					if now.After(v) {
						Logger.Info("expriring timeout", zap.String("user", k))
						delete(timeouts.data, k)
					}
				}
				timeouts.Unlock()
			}
		}
	}()
}
