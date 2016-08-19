package bot

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
