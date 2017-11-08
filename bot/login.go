package bot

import (
	"fmt"
	"os"
	"strings"

	"github.com/nlopes/slack"
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
	if home := os.Getenv("SERVICE_DIR"); home == "" {
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
	return 0 < DB.Exec("UPDATE logins SET member = ? WHERE code = ? LIMIT 1", m.User, strings.ToLower(m.Msg.Text)[:max]).RowsAffected
}

func handleLogin(m *slack.MessageEvent) bool {
	if home := os.Getenv("SERVICE_DIR"); home != "" {
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
