package bot

import (
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"strings"
	"time"
)

const channelCommand = "!channel"
//const memberCategoryID = "556671265650507812" //BearKhan
const memberCategoryID = "556688616911536129" //FTS

// !channel channel_name
func (d *DiscordAPI) tempChannelCommandHandler(s *discordgo.Session, event *discordgo.MessageCreate) {
	fields := strings.Fields(event.Content)
	if fields[0] != channelCommand {
		return
	}

	channelData := discordgo.GuildChannelCreateData{
		Name:     fields[1],
		ParentID: memberCategoryID,
		Type:     discordgo.ChannelTypeGuildText,
	}
	_, err := d.discord.GuildChannelCreateComplex(d.Config.GuildId, channelData)
	if err != nil {
		Logger.Error("unable to create member channel", zap.String("channel_name", fields[1]), zap.Error(err))
	}

}

func (d *DiscordAPI) mindTempChannels() {
	tick := time.Tick(1 * time.Hour) // 1 hour mind, because this could have been started at any time
	for {
		select {
		case <-tick:
			channels := d.guildChannels()
			memberChannels := []*Channel{}

			// get channels of member channels category
			for _, category := range channels.Categories {
				if category.ID == memberCategoryID {
					memberChannels = category.Channels
					break
				}
			}

			for _, channel := range memberChannels {
				// get channel data
				ch, err := d.discord.Channel(channel.ID)
				if err != nil {
					Logger.Error("could not get channel", zap.String("channel_id", channel.ID), zap.Error(err))
					continue
				}

				// skip non text channels
				if ch.Type != discordgo.ChannelTypeGuildText {
					continue
				}

				// get last message in channel
				lastMessage, err := d.discord.ChannelMessage(ch.ID, ch.LastMessageID)
				if err != nil {
					// no last message means a new channel
					continue
				}

				// get time of last message
				lastMessageTime, err := lastMessage.Timestamp.Parse()
				if err != nil {
					Logger.Error("unable to parse timestamp. check for API changes", zap.String("message", string(lastMessage.Timestamp)), zap.Error(err))
					continue
				}

				// if more than 7 days, delete
				if time.Since(lastMessageTime) > (time.Hour * 24 * 7) {
					_, err := d.discord.ChannelDelete(ch.ID)
					if err != nil {
						Logger.Error("unable to delete member channel", zap.String("channel", ch.ID), zap.Error(err))
						continue
					} else {
						Logger.Info("deleted channel", zap.String("channel", ch.ID), zap.String("name", ch.Name))
					}

				}
			}
		}
	}
}
