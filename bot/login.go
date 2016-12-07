package bot

import (
	"fmt"
	"os"
	"time"

	"github.com/nlopes/slack"
)

var LoginLink = ""

type AuthTokenGeneratorFunc func(string) []string

var AuthTokenGenerator = func(s string) []string {
	return nil
}

func handleLogin(m *slack.MessageEvent) bool {
	if m.Msg.Text != "login" {
		return false
	}

	if AuthTokenGenerator == nil {
		return false
	}

	var linkText = "Login with this link"
	if home := os.Getenv("SERVICE_DIR"); home != "" {
		linkText = "Dev login with this link"
	}

	var msg = fmt.Sprintf(
		"<%sapi/v0/login?w=%s&t=%s|%s>",
		LoginLink,
		m.Msg.User,
		AuthTokenGenerator(m.Msg.User)[0],
		linkText)

	for i := 0; i < 5; i++ {
		_, _, err := rtm.PostMessage(
			m.Channel,
			msg,
			slack.PostMessageParameters{
				Text:        msg,
				AsUser:      true,
				UnfurlLinks: true,
				UnfurlMedia: true,
				IconEmoji:   ":link:",
			})
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i) * time.Second)
	}
	return true
}
