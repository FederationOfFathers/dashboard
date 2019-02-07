package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/FederationOfFathers/dashboard/environment"
	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

var LoginLink = ""

type AuthTokenGeneratorFunc func(string) []string

var AuthTokenGenerator = func(s string) []string {
	return nil
}

func SendDevLogin(user string) {
	var linkText = "Dev login with this link"
	var msg = fmt.Sprintf(
		"<%sapi/v0/login?w=%s&t=%s|%s>",
		LoginLink,
		user,
		AuthTokenGenerator(user)[0],
		linkText)
	fofbotMessage <- sendMessage{user, msg}
}

func handleDevLogin(m *slack.MessageEvent) bool {
	if !environment.IsDev {
		return false
	}

	if len(strings.TrimSpace(m.Msg.Text)) != 9 {
		return false
	}
	if strings.ToLower(strings.TrimSpace(m.Msg.Text)) != "dev login" {
		return false
	}

	if AuthTokenGenerator == nil {
		return false
	}

	SendDevLogin(m.Msg.User)
	return true
}

func SendLogin(user string) {
	var linkText = "Login with this link"
	var msg = fmt.Sprintf(
		"<%sapi/v0/login?w=%s&t=%s|%s>",
		LoginLink,
		user,
		AuthTokenGenerator(user)[0],
		linkText)
	fofbotMessage <- sendMessage{user, msg}
}

func handleLoginCode(m *slack.MessageEvent) bool {
	max := len(m.Msg.Text)
	if max > 190 {
		max = 190
	}
	var handled bool
	member, err := DB.MemberBySlackID(m.User)
	if err != nil {
		Logger.Error("no member found", zap.String("slackId", m.User), zap.Error(err))
		return false
	}
	if dbErr := DB.Exec("UPDATE logins SET member = ?, member_id = ? WHERE code = ? LIMIT 1", m.User, member.ID, strings.ToLower(m.Msg.Text)[:max]).Error; dbErr != nil {
		Logger.Error("unable to update login", zap.String("code", m.Text), zap.String("user", m.User), zap.Int("id", member.ID), zap.Error(dbErr))
		handled = false
	} else {
		handled = true
	}

	if handled {
		rtm.SendMessage(&slack.OutgoingMessage{
			ID:      int(time.Now().UnixNano()),
			Channel: m.Msg.Channel,
			Text:    "Your login should complete momentarily",
			Type:    "message",
		})
	}
	return handled
}

func handleLogin(m *slack.MessageEvent) bool {
	if !environment.IsProd {
		return false
	}
	if len(strings.TrimSpace(m.Msg.Text)) != 5 {
		return false
	}
	if strings.ToLower(strings.TrimSpace(m.Msg.Text)) != "login" {
		return false
	}

	if AuthTokenGenerator == nil {
		return false
	}
	SendLogin(m.Msg.User)
	return true
}
