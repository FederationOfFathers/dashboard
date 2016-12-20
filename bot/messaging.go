package bot

import "github.com/nlopes/slack"

type sendMessage struct {
	to   string
	text string
}

var postMessage = make(chan sendMessage, 64)

func SendMessage(to, message string) {
	postMessage <- sendMessage{
		to:   to,
		text: message,
	}
}

func SendMessageAs(to, message, as string) {
	go PostMessage(
		to,
		message,
		slack.PostMessageParameters{
			Username:    as,
			AsUser:      true,
			UnfurlLinks: true,
			UnfurlMedia: true,
		},
	)
}

func PostMessage(to, message string, params slack.PostMessageParameters) error {
	_, _, err := api.PostMessage(to, message, params)
	return err
}
