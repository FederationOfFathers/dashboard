package bot

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/messaging"
	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

var api *slack.Client
var rtm *slack.RTM
var connection *slack.Info
var connected bool
var token string
var Logger *zap.Logger
var StartupNotice = false
var DB *db.DB

// LogLevel sets the logging verbosity for the package
var LogLevel = zap.InfoLevel

type SlackAPI struct {
	Token                 string
	Slack                 *slack.Client
	streamNoticeChannelId string
}

func NewSlackAPI(token string, streamChannelId string) SlackAPI {
	return SlackAPI{
		Token: token,
		streamNoticeChannelId: streamChannelId,
	}
}

func (s *SlackAPI) Connect() {
	if messagingClient != nil {
		s.Slack = messagingClient
	}
}

func (s SlackAPI) Shutdown() {
	// nothing to do here, but maybe one day
}

// SlackConnect gets the whole party stated
func SlackConnect(slackToken string) error {
	bridge.Data.Slack = data
	bridge.SendMessage = SendMessage
	bridge.PostMessage = PostMessage
	data.load()
	api = slack.New(slackToken)
	populateLists()
	if len(data.Users) < 1 {
		return ErrSlackAPIUnresponsive
	}
	token = slackToken
	rtm = api.NewRTM()
	// rtm.SetDebug(true)
	go mindLists()
	go rtm.ManageConnection()
	go func() {
		for {
			if err := mindSlack(); err != nil {
				Logger.Error("error Minding slack", zap.Error(err))
				time.Sleep(30 * time.Second)
			}
		}
	}()
	messagingClient = &rtm.Client
	if MessagingKey != "" {
		Logger.Warn("Using special key for fofbot messaging", zap.String("key", MessagingKey))
		messagingClient = slack.New(MessagingKey)
	} else {
		Logger.Warn("Using default client for fofbot messaging")
	}
	if StartupNotice {
		if home := os.Getenv("SERVICE_DIR"); home != "" {
			SendMessage("#-fof-dashboard", "Dev Dashboard starting up...")
		} else {
			SendMessage("#-fof-dashboard", "Production Dashboard starting up...")
		}
	}
	return nil
}

func mindSlack() error {
	for {
		select {
		case msg := <-fofbotMessage:
			_, _, err := messagingClient.PostMessage(msg.to, msg.text, slack.PostMessageParameters{
				AsUser:      true,
				UnfurlLinks: true,
				UnfurlMedia: true,
			})
			if err != nil {
				Logger.Error("client.PostMessage failed", zap.Error(err))
			}
		case msg := <-postMessage:
			_, _, err := rtm.PostMessage(msg.to, msg.text, slack.PostMessageParameters{
				AsUser:      true,
				UnfurlLinks: true,
				UnfurlMedia: true,
			})
			if err != nil {
				Logger.Error("rtm.PostMessage failed", zap.Error(err))
			}
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			// Connecting and Disconnecting
			case *slack.ConnectedEvent:
				Logger.Debug("slack.ConnectedEvent", zap.Int("count", ev.ConnectionCount))
				connection = ev.Info
				connected = true
			case *slack.DisconnectedEvent:
				connected = false
				Logger.Debug("slack.DisconnectedEvent", zap.Bool("intentional", ev.Intentional))

			// Groups
			case *slack.GroupCloseEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.GroupCloseEvent", zap.Bool("handled", true))
			case *slack.GroupJoinedEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.GroupJoinedEvent", zap.Bool("handled", true))
			case *slack.GroupLeftEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.GroupLeftEvent", zap.Bool("handled", true))
			case *slack.GroupRenameEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.GroupRenameEvent", zap.Bool("handled", true))

			// Instant Messages
			case *slack.IMCloseEvent:
				Logger.Debug("slack.IMCloseEvent", zap.Bool("handled", false))
			case *slack.IMCreatedEvent:
				Logger.Debug("slack.IMCreatedEvent", zap.Bool("handled", false))
			case *slack.IMHistoryChangedEvent:
				Logger.Debug("slack.IMHistoryChangedEvent", zap.Bool("handled", false))
			case *slack.IMMarkedEvent:
				Logger.Debug("slack.IMMarkedEvent", zap.Bool("handled", false))
			case *slack.IMOpenEvent:
				Logger.Debug("slack.IMOpenEvent", zap.Bool("handled", false))

			// Channels
			case *slack.ChannelHistoryChangedEvent:
				Logger.Debug("slack.ChannelHistoryChangedEvent", zap.Bool("handled", false))
			case *slack.ChannelArchiveEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.ChannelArchiveEvent", zap.Bool("handled", true))
			case *slack.ChannelCreatedEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.ChannelCreatedEvent", zap.Bool("handled", true))
			case *slack.ChannelDeletedEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.ChannelDeletedEvent", zap.Bool("handled", true))
			case *slack.ChannelInfoEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.ChannelInfoEvent", zap.Bool("handled", true))
			case *slack.ChannelJoinedEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.ChannelJoinedEvent", zap.Bool("handled", true))
			case *slack.ChannelLeftEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.ChannelLeftEvent", zap.Bool("handled", true))
			case *slack.ChannelRenameEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.ChannelRenameEvent", zap.Bool("handled", true))
			case *slack.ChannelUnarchiveEvent:
				UpdateRequest <- struct{}{}
				Logger.Debug("slack.ChannelUnarchiveEvent", zap.Bool("handled", true))

			// Users
			case *slack.MessageEvent:
				go func(ev *slack.MessageEvent) {
					switch strings.ToLower(ev.Channel[:1]) {
					case "d":
						if !handleDirectMessage(ev) {
							Logger.Info("slack.MessageEvent",
								zap.Bool("handled", false),
								zap.String("type", "direct_message"),
								zap.String("user", ev.Msg.User),
								zap.String("channel", ev.Msg.Channel),
								zap.String("message", ev.Msg.Text),
								zap.String("debug", fmt.Sprintf("%#v", ev)))
						}
					case "g":
						if !handleGroupMessage(ev) {
							Logger.Debug("slack.MessageEvent",
								zap.Bool("handled", false),
								zap.String("type", "group_message"),
								zap.String("user", ev.Msg.User),
								zap.String("channel", ev.Msg.Channel),
								zap.String("message", ev.Msg.Text))
						}
					case "c":
						if !handleChannelMessage(ev) {
							Logger.Debug("slack.MessageEvent",
								zap.Bool("handled", false),
								zap.String("type", "channel_message"),
								zap.String("user", ev.Msg.User),
								zap.String("channel", ev.Msg.Channel),
								zap.String("message", ev.Msg.Text))
						}
					default:
						Logger.Debug("slack.MessageEvent",
							zap.Bool("handled", false),
							zap.String("type", "unknown"),
							zap.String("raw", fmt.Sprintf("%#v", *ev)),
							zap.String("user", ev.Msg.User),
							zap.String("channel", ev.Msg.Channel),
							zap.String("message", ev.Msg.Text))
					}
				}(ev)
			case *slack.PresenceChangeEvent:
				Logger.Debug("slack.PresenceChangeEvent",
					zap.Bool("handled", false),
					zap.String("user", ev.User),
					zap.String("type", ev.Type),
					zap.String("presence", ev.Presence))
			case *slack.UserChangeEvent:
				Logger.Debug("slack.UserChangeEvent", zap.Bool("handled", false))
			case *slack.UserTypingEvent:
				// Not necessary to let us know. thanks :)
				// Logger.Debug("slack.UserTypingEvent", zap.Bool("handled", false))
			case *slack.DNDUpdatedEvent:
				Logger.Debug("slack.DNDUpdatedEvent", zap.Bool("handled", false))
			case *slack.PrefChangeEvent:
				Logger.Debug("slack.PrefChangeEvent", zap.Bool("handled", false))
			case *slack.TeamJoinEvent:
				Logger.Debug("slack.TeamJoinEvent", zap.Bool("handled", false))

			// Files
			case *slack.FileCreatedEvent:
				Logger.Debug("slack.FileCreatedEvent", zap.Bool("handled", false))
			case *slack.FileDeletedEvent:
				Logger.Debug("slack.FileDeletedEvent", zap.Bool("handled", false))
			case *slack.FilePrivateEvent:
				Logger.Debug("slack.FilePrivateEvent", zap.Bool("handled", false))
			case *slack.FilePublicEvent:
				Logger.Debug("slack.FilePublicEvent", zap.Bool("handled", false))
			case *slack.FileSharedEvent:
				Logger.Debug("slack.FileSharedEvent", zap.Bool("handled", false))
			case *slack.FileUnsharedEvent:
				Logger.Debug("slack.FileUnsharedEvent", zap.Bool("handled", false))

			// Errors
			case *slack.UnmarshallingErrorEvent:
				// new and/or unhandled message type
				continue
			case *slack.OutgoingErrorEvent:
				return ev
			case *slack.AckErrorEvent:
				return ev
			case *slack.RTMError:
				return ev
			case *slack.InvalidAuthEvent:
				Logger.Fatal("Slack got InvalidAuthEvent")
			}
		}
	}
}

func GroupInvite(groupID, userID string) error {
	_, _, err := api.InviteUserToGroup(groupID, userID)
	return err
}

func ChannelInvite(channelID, userID string) error {
	_, err := api.InviteUserToChannel(channelID, userID)
	return err
}

func GroupKick(groupID, userID string) error {
	return api.KickUserFromGroup(groupID, userID)
}

func ChannelKick(channelID, userID string) error {
	return api.KickUserFromChannel(channelID, userID)
}

func (s SlackAPI) PostStreamMessage(sm messaging.StreamMessage) error {
	if s.Slack == nil {
		return fmt.Errorf("slack API not connected")
	}

	messageParams := slack.NewPostMessageParameters()
	message := fmt.Sprintf("*%s is live!* - %s", sm.Username, sm.URL)
	messageParams.AsUser = true
	messageParams.Parse = "full"
	messageParams.LinkNames = 1
	messageParams.UnfurlMedia = true
	messageParams.UnfurlLinks = false
	messageParams.EscapeText = false
	messageParams.Attachments = append(messageParams.Attachments, slack.Attachment{
		Fallback:   message,
		Color:      sm.PlatformColor,
		Title:      fmt.Sprintf("%s is live!", sm.Username),
		TitleLink:  sm.URL,
		ThumbURL:   sm.UserLogo,
		Footer:     fmt.Sprintf("%s | %s", sm.Platform, sm.Timestamp),
		FooterIcon: sm.PlatformLogo,
		Fields: []slack.AttachmentField{
			{
				Title: "Game",
				Value: fmt.Sprintf("%s - %s", sm.Game, sm.Description),
				Short: false,
			},
		},
	})

	_, _, err := s.Slack.PostMessage(s.streamNoticeChannelId, "", messageParams)
	return err
}
