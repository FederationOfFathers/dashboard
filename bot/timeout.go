package bot

import (
	"strings"
	"sync"
	"time"

	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
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
			logger.Warn("user not admin... timeout")
			return false
		}
		guess := strings.Trim(parsed[2], "<>@:#-=+!@#$%^&*(){}[]\\|<>?,./")
		if d, err := time.ParseDuration(parsed[3]); err != nil {
			logger.Error(
				"error parsing duration",
				zap.String("string", guess),
				zap.Error(err))
		} else {
			logger.Warn("Adding timeout", zap.String("user", guess), zap.Duration("duration", d))
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
		logger.Error(
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
						logger.Info("expriring timeout", zap.String("user", k))
						delete(timeouts.data, k)
					}
				}
				timeouts.Unlock()
			}
		}
	}()
}
