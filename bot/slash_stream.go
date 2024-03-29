package bot

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/bwmarrin/discordgo"
	"github.com/gocolly/colly/v2"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"google.golang.org/api/youtube/v3"
)

const (
	twitchStreamLinkFmt  = "https://www.twitch.tv/%s"
	youtubeStreamLinkFmt = "https://www.youtube.com/channel/%s"
)

// registerSlashStream regsiters the /stream add/remove commands for the bot
func (d *DiscordAPI) registerSlashStream() {

	streamCommand := &discordgo.ApplicationCommand{
		Name:        "stream",
		Description: "Use to register/unregister streams to be announced in the on-air channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "add",
				Description: "Adds a stream to be monitored and announced in the on-air channel",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "stream",
						Description: "link to your stream (http://twitch.tv/my_username)",
						Required:    true,
						Type:        discordgo.ApplicationCommandOptionString,
					},
				},
			},
			{
				Name:        "remove",
				Description: "use to remove any stream linked to your profile",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}

	if _, err := d.discord.ApplicationCommandCreate(d.discord.State.User.ID, d.Config.GuildId, streamCommand); err != nil {
		Logger.With(zap.Error(err)).Error("unable to register slash commands")
	} else {
		Logger.Info("Discord slash commands registered")
	}

}

// slashStreamHandler handles the initial /stream add/remove commands, not the button interactions
func (d *DiscordAPI) slashStreamHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {

	commandData := i.ApplicationCommandData()
	Logger.With(zap.String("name", commandData.Name), zap.String("id", commandData.ID), zap.Any("options", commandData.Options)).Info("slash command")

	badOptionResponse := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "use `/stream add <stream_link>` or `/stream remove`",
			Flags:   64,
		},
	}

	// check option for add / remove
	if len(commandData.Options) == 0 {
		s.InteractionRespond(i.Interaction, badOptionResponse)
		return
	}

	switch commandData.Options[0].Name {
	case "add":

		streamLink := commandData.Options[0].Options[0].StringValue()
		var streamType string
		var streamUser string
		var streamID string
		var channelLink string

		// determine stream type and username
		if strings.Contains(streamLink, "twitch.tv/") || strings.Contains(streamLink, "twitch.com/") {
			streamType = "twitch"
			streamUser = streamLink[strings.LastIndex(streamLink, "/")+1:]
			streamID = streamUser
			channelLink = fmt.Sprintf(twitchStreamLinkFmt, streamID)
		} else if strings.Contains(streamLink, "youtube.com/") {
			if d.yt == nil {
				s.InteractionRespond(i.Interaction, badOptionResponse)
				return
			}
			streamType = "youtube"
			ytChannel, err := d.getYoutubeChannelFromURL(streamLink)
			if err != nil {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "OOPS! We had some trouble processing your link.",
						Flags:   64,
					},
				})
				Logger.With(zap.String("youtube_url", streamLink), zap.Error(err)).Error("parsing youtube channel stream failed")
				return
			}
			streamUser = ytChannel.BrandingSettings.Channel.Title
			streamID = ytChannel.Id
			channelLink = fmt.Sprintf(youtubeStreamLinkFmt, streamID)
		}

		if streamUser == "" || streamType == "" {
			s.InteractionRespond(i.Interaction, badOptionResponse)
			return
		}

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   64,
				Content: fmt.Sprintf("Do you want to add the stream for %s? \n%s\n(This will replace any current stream you may have already added)", streamUser, channelLink),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "yes",
								Style:    discordgo.PrimaryButton,
								CustomID: fmt.Sprintf("stream:add:confirm:%s:%s", streamType, streamID),
							},
							discordgo.Button{
								Label:    "no",
								Style:    discordgo.SecondaryButton,
								CustomID: fmt.Sprintf("stream:add:cancel"),
							},
						},
					},
				},
			},
		})
		if err != nil {
			Logger.With(zap.Error(err)).Error("response failed")
		}

	case "remove":
		// removes all streams for users
		m, err := DB.MemberByDiscordID(i.Member.User.ID)
		if err != nil {
			// d.sendInteractionResponse(i, "sorry, i couldn't find your message")
			Logger.With(zap.Error(err)).Error("unable to find member")
			break
		}

		Logger.With(zap.String("discordID", i.Member.User.ID), zap.Int("memberID", m.ID)).Info("removing streams")

		stream, err := DB.StreamByMemberID(m.ID)
		if err != nil && err != gorm.ErrRecordNotFound {

			Logger.With(zap.Error(err), zap.Int("memberID", m.ID)).Error("could not retrieve member stream")
			return

		}

		if stream.MemberID == 0 {
			stream.MemberID = m.ID
		}

		stream.Twitch = ""
		stream.Youtube = ""

		if err := stream.Save(); err != nil {
			Logger.With(zap.Error(err)).Error("unable to remove streams")
			break
		}
		Logger.Info("removing stream")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Your stream has been removed and will no longer be shared when you go live",
				Flags:   64,
			},
		})
	default:
		s.InteractionRespond(i.Interaction, badOptionResponse)
	}
	// if add, get the user, verify the link and upset
	// if remove, get the user, and remove the stream
	// s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseChannelMessageWithSource,
	// 	Data: &discordgo.InteractionApplicationCommandResponseData{Content: "OK"},
	// })

}

func (d *DiscordAPI) getYoutubeChannelFromURL(url string) (*youtube.Channel, error) {

	channelList := d.yt.Channels.List([]string{"id", "brandingSettings"})
	if strings.Contains(url, "/channel/") {
		id := url[strings.LastIndex(url, "/")+1:]
		channelList = channelList.Id(id)
	} else if strings.Contains(url, "/user/") {
		username := url[strings.LastIndex(url, "/")+1:]
		channelList = channelList.ForUsername(username)
	} else if strings.Contains(url, "/c/") {

		// there is currently no API endpoint to get a channel by the custom
		// URL, so we need to do some HTML scraping to get and verify the channel
		channelUrl := getCanonicalURLForCustomURL(url)
		if channelUrl == "" {
			return nil, fmt.Errorf("could not find a valid YouTube channel")
		}

		id := channelUrl[strings.LastIndex(channelUrl, "/")+1:]
		Logger.With(zap.String("url", channelUrl), zap.String("id", id)).Debug("channel URL found")
		channelList = channelList.Id(id)
	} else {
		return nil, fmt.Errorf("unable to parse the URL")
	}

	resp, err := channelList.Do()
	if err != nil {
		return nil, err
	}

	if len(resp.Items) > 0 {
		return resp.Items[0], nil
	}

	return nil, fmt.Errorf("could not find a matching youtube channel")

}

func getCanonicalURLForCustomURL(url string) string {
	var canonicalUrl string
	c := colly.NewCollector()

	c.OnHTML("link[rel]", func(e *colly.HTMLElement) {
		rel := e.Attr("rel")
		if rel == "canonical" {
			canonicalUrl = e.Attr("href")
		}
	})

	if err := c.Visit(url); err != nil {
		Logger.With(zap.String("url", url), zap.Error(err)).Error("unable to visit YouTube url")
	}

	return canonicalUrl
}

// slashStreamComponentHandler handles the component interactions, such as button clicks for confirmation
func (d *DiscordAPI) slashStreamComponentHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {

	// stream:add|remove:confirm|cancel:?type:?user
	customID := i.MessageComponentData().CustomID
	Logger.With(zap.String("customID", customID)).Debug("START slashStreamComponentHandler")
	componentParts := strings.Split(customID, ":")

	switch componentParts[2] {
	case "confirm":
		streamType := componentParts[3]
		streamUsername := componentParts[4]

		// get or create the member record
		m, err := DB.MemberByDiscordID(i.Member.User.ID)
		if err != nil {
			if err == gorm.ErrRecordNotFound || err == sql.ErrNoRows {
				Logger.Info("adding new member")

				// new member
				m = db.NewMember(DB)
				m.Discord = i.Member.User.ID
				m.Name = i.Member.Nick
				m.Save()
				newM, err := DB.MemberByDiscordID(i.Member.User.ID)
				if err != nil {
					Logger.With(zap.Error(err)).Error("unable to retrieve newly created member")
					return
				}
				m = newM
			}
			Logger.With(zap.Error(err)).Error("unable to find member data")
			return
		}

		stream, err := DB.StreamByMemberID(m.ID)
		if err != nil && err != gorm.ErrRecordNotFound {

			Logger.With(zap.Error(err), zap.Int("memberID", m.ID)).Error("could not retrieve member stream")
			return

		}

		if stream.MemberID == 0 {
			stream.MemberID = m.ID
		}

		switch streamType {
		case "twitch":
			stream.Twitch = streamUsername
			stream.Youtube = ""
		case "youtube":
			stream.Youtube = streamUsername
			stream.Twitch = ""
		default:
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "hmm, something didn't go right...sorry! try again if you must",
					Flags:   64,
				},
			})
			Logger.With(zap.String("button_id", customID)).Error("unknown stream type option")
			return
		}

		if err := stream.Save(); err != nil {
			Logger.With(zap.Error(err), zap.Any("stream", stream)).Error("could not save stream data")
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "OK, your stream has been added and will now be posted in the designated channel when you are live",
				Flags:   64,
			},
		})
	case "cancel":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "OK, request canceled",
				Flags:   64,
			},
		})
	}

}
