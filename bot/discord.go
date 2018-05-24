package bot

import (
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"fmt"
	"github.com/FederationOfFathers/dashboard/messaging"
)

var logger = messaging.Logger

type DiscordAPI struct {
	Token string
	discord *discordgo.Session
	streamNoticeChannelId string
}

func NewDiscordAPI(token string, streamChannelId string) DiscordAPI {
	return DiscordAPI{
		Token: token,
		streamNoticeChannelId: streamChannelId,
	}
}

// Needs to be called before any other API function work
func (d *DiscordAPI) Connect() {
	dg, err := discordgo.New("Bot " + d.Token)
	if err != nil {
		logger.Error("Unable to create discord connection", zap.Error(err))
		return
	}

	d.discord = dg
	dg.Open()
}

// Needs to be called to disconnect from discord
func (d *DiscordAPI) Shutdown() {
	d.discord.Close()
}

func (d DiscordAPI) PostStreamMessage(sm messaging.StreamMessage) error {
	if d.discord == nil {
		return fmt.Errorf("discord API not connected")
	}
	author := discordgo.MessageEmbedAuthor{
		Name: fmt.Sprintf("%s is live!", sm.Username),
	}
	thumbnail := discordgo.MessageEmbedThumbnail{
		URL: sm.UserLogo,
	}
	footer := discordgo.MessageEmbedFooter{
		Text: sm.Platform,
		IconURL: sm.PlatformLogo,
	}
	messageEmbed := discordgo.MessageEmbed{
		Description: fmt.Sprintf("**Game:** %s\n%s\n%s", sm.Game, sm.Description, sm.URL),
		Color: sm.PlatformColorInt,
		URL: sm.URL,
		Author: &author,
		Thumbnail: &thumbnail,
		Footer: &footer,
	}
	_, err := d.discord.ChannelMessageSendEmbed(d.streamNoticeChannelId, &messageEmbed)
	return err
}