package bot

import (
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const channelCommand = "!channel"

// !channel channel_name
func (d *DiscordAPI) tempChannelCommandHandler(s *discordgo.Session, event *discordgo.MessageCreate) {

	noResponses := []string{
		"Sorry, I don't do that any more.",
		"You can't make me.",
		"Just what do you think you're doing, Dave?",
		"Sorry, I'm taking a shit and posting about it on a channel that doesn't exist anymore",
		"Remember when I used to be on Slack? HA!",
		"I'm telling Meow",
		"Thank you! Beginning self destruct...",
		"That's not for you",
		"You are not authorized to access this area.",
		"Shall we play a game? Oh... wait... I don't know how to play games...",
		"End Of Line",
	}

	rand.Seed(time.Now().UnixNano())
	randResponse := rand.Intn(len(noResponses) - 1)
	s.ChannelMessageSendReply(event.ChannelID, noResponses[randResponse], event.Reference())

}

func (d DiscordAPI) removeReaction(channelID, messageID, emojiID, userID string) {
	// remove the users reaction. If the add/remove failed, they can click it again to re-trigger
	err := d.discord.MessageReactionRemove(
		channelID,
		messageID,
		emojiID,
		userID,
	)

	if err != nil {
		Logger.Error("could not remove reaction from message",
			zap.String("user", userID),
			zap.String("emoji", emojiID),
			zap.Error(err))
	}
}
