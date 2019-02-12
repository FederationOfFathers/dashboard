package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

func handleDiscordLink(m *slack.MessageEvent) bool {

	msg := strings.SplitN(m.Msg.Text, " ", 2)

	if strings.ToLower(msg[0]) == "discord" {
		var outgoingText string
		if discordApi != nil {
			discordMember := msg[1]
			outgoingText = syncDiscordFromMsg(discordMember, m)
		}

		outMsg := &slack.OutgoingMessage{
			ID:      int(time.Now().UnixNano()),
			Channel: m.Msg.Channel,
			Text:    outgoingText,
			Type:    "message",
		}
		rtm.SendMessage(outMsg)

		return true
	}

	return false

}

func syncDiscordFromMsg(discordMember string, m *slack.MessageEvent) string {
	discordID, username := discordApi.FindIDByUsername(discordMember)
	if discordID == "" {
		Logger.Error("Could not find Discord member", zap.String("username", discordMember))
		return "Check your username again. Should be in the format `Username#0000`"
	}

	member, err := DB.MemberBySlackID(m.Msg.User)
	if err != nil {
		Logger.Error("Could not find slack user", zap.String("slack user", m.Msg.User))
		return "Your acocunt isn't in Team Tool yet"
	}
	member.Discord = discordID
	member.Name = username

	if err := member.Save(); err != nil {
		Logger.Error("Unable to save discord to member", zap.String("discord", discordID), zap.String("slack", m.Msg.User))
		return "There was a problem saving your information. Try again later."
	}

	return fmt.Sprintf("Discord user `%s` Has been connected to your account", discordMember)

}
