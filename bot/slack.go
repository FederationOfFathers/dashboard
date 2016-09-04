package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
)

var api *slack.Client
var rtm *slack.RTM
var connection *slack.Info
var connected bool
var token string
var logger = zap.NewJSON().With(zap.String("module", "bot"))

// LogLevel sets the logging verbosity for the package
var LogLevel = zap.InfoLevel

// SlackConnect gets the whole party stated
func SlackConnect(slackToken string) error {
	bridge.Data.Slack = data
	bridge.SendMessage = SendMessage
	bridge.PostMessage = PostMessage
	data.load()
	logger.SetLevel(LogLevel)
	api = slack.New(slackToken)
	populateLists()
	if len(data.Users) < 1 {
		return ErrSlackAPIUnresponsive
	}
	token = slackToken
	rtm = api.NewRTM()
	go mindLists()
	go rtm.ManageConnection()
	go func() {
		for {
			if err := mindSlack(); err != nil {
				logger.Error("error Minding slack", zap.Error(err))
				time.Sleep(30 * time.Second)
			}
		}
	}()
	return nil
}

func mindSlack() error {
	for {
		select {
		case msg := <-postMessage:
			_, _, err := rtm.PostMessage(msg.to, msg.text, slack.PostMessageParameters{
				AsUser:      true,
				UnfurlLinks: true,
				UnfurlMedia: true,
			})
			if err != nil {
				logger.Error("rtm.PostMessage failed", zap.Error(err))
			}
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			// Connecting and Disconnecting
			case *slack.ConnectedEvent:
				logger.Debug("slack.ConnectedEvent", zap.Int("count", ev.ConnectionCount))
				connection = ev.Info
				connected = true
			case *slack.DisconnectedEvent:
				connected = false
				logger.Debug("slack.DisconnectedEvent", zap.Bool("intentional", ev.Intentional))

			// Groups
			case *slack.GroupCloseEvent:
				logger.Debug("slack.GroupCloseEvent", zap.Bool("handled", false))
			case *slack.GroupJoinedEvent:
				logger.Debug("slack.GroupJoinedEvent", zap.Bool("handled", false))
			case *slack.GroupLeftEvent:
				logger.Debug("slack.GroupLeftEvent", zap.Bool("handled", false))
			case *slack.GroupRenameEvent:
				logger.Debug("slack.GroupRenameEvent", zap.Bool("handled", false))

			// Instant Messages
			case *slack.IMCloseEvent:
				logger.Debug("slack.IMCloseEvent", zap.Bool("handled", false))
			case *slack.IMCreatedEvent:
				logger.Debug("slack.IMCreatedEvent", zap.Bool("handled", false))
			case *slack.IMHistoryChangedEvent:
				logger.Debug("slack.IMHistoryChangedEvent", zap.Bool("handled", false))
			case *slack.IMMarkedEvent:
				logger.Debug("slack.IMMarkedEvent", zap.Bool("handled", false))
			case *slack.IMOpenEvent:
				logger.Debug("slack.IMOpenEvent", zap.Bool("handled", false))

			// Channels
			case *slack.ChannelArchiveEvent:
				logger.Debug("slack.ChannelArchiveEvent", zap.Bool("handled", false))
			case *slack.ChannelCreatedEvent:
				logger.Debug("slack.ChannelCreatedEvent", zap.Bool("handled", false))
			case *slack.ChannelDeletedEvent:
				logger.Debug("slack.ChannelDeletedEvent", zap.Bool("handled", false))
			case *slack.ChannelHistoryChangedEvent:
				logger.Debug("slack.ChannelHistoryChangedEvent", zap.Bool("handled", false))
			case *slack.ChannelInfoEvent:
				logger.Debug("slack.ChannelInfoEvent", zap.Bool("handled", false))
			case *slack.ChannelJoinedEvent:
				logger.Debug("slack.ChannelJoinedEvent", zap.Bool("handled", false))
			case *slack.ChannelLeftEvent:
				logger.Debug("slack.ChannelLeftEvent", zap.Bool("handled", false))
			case *slack.ChannelRenameEvent:
				logger.Debug("slack.ChannelRenameEvent", zap.Bool("handled", false))
			case *slack.ChannelUnarchiveEvent:
				logger.Debug("slack.ChannelUnarchiveEvent", zap.Bool("handled", false))

			// Users
			case *slack.MessageEvent:
				go func(ev *slack.MessageEvent) {
					switch strings.ToLower(ev.Channel[:1]) {
					case "d":
						if !handleDirectMessage(ev) {
							logger.Info("slack.MessageEvent",
								zap.Bool("handled", false),
								zap.String("type", "direct_message"),
								zap.String("user", ev.Msg.User),
								zap.String("channel", ev.Msg.Channel),
								zap.String("message", ev.Msg.Text),
								zap.String("debug", fmt.Sprintf("%#v", ev)))
						}
					case "g":
						if !handleGroupMessage(ev) {
							logger.Debug("slack.MessageEvent",
								zap.Bool("handled", false),
								zap.String("type", "group_message"),
								zap.String("user", ev.Msg.User),
								zap.String("channel", ev.Msg.Channel),
								zap.String("message", ev.Msg.Text))
						}
					case "c":
						if !handleChannelMessage(ev) {
							logger.Debug("slack.MessageEvent",
								zap.Bool("handled", false),
								zap.String("type", "channel_message"),
								zap.String("user", ev.Msg.User),
								zap.String("channel", ev.Msg.Channel),
								zap.String("message", ev.Msg.Text))
						}
					default:
						logger.Debug("slack.MessageEvent",
							zap.Bool("handled", false),
							zap.String("type", "unknown"),
							zap.String("raw", fmt.Sprintf("%#v", *ev)),
							zap.String("user", ev.Msg.User),
							zap.String("channel", ev.Msg.Channel),
							zap.String("message", ev.Msg.Text))
					}
				}(ev)
			case *slack.PresenceChangeEvent:
				logger.Debug("slack.PresenceChangeEvent",
					zap.Bool("handled", false),
					zap.String("user", ev.User),
					zap.String("type", ev.Type),
					zap.String("presence", ev.Presence))
			case *slack.UserChangeEvent:
				logger.Debug("slack.UserChangeEvent", zap.Bool("handled", false))
			case *slack.UserTypingEvent:
				logger.Debug("slack.UserTypingEvent", zap.Bool("handled", false))
			case *slack.DNDUpdatedEvent:
				logger.Debug("slack.DNDUpdatedEvent", zap.Bool("handled", false))
			case *slack.PrefChangeEvent:
				logger.Debug("slack.PrefChangeEvent", zap.Bool("handled", false))
			case *slack.TeamJoinEvent:
				logger.Debug("slack.TeamJoinEvent", zap.Bool("handled", false))

			// Files
			case *slack.FileCreatedEvent:
				logger.Debug("slack.FileCreatedEvent", zap.Bool("handled", false))
			case *slack.FileDeletedEvent:
				logger.Debug("slack.FileDeletedEvent", zap.Bool("handled", false))
			case *slack.FilePrivateEvent:
				logger.Debug("slack.FilePrivateEvent", zap.Bool("handled", false))
			case *slack.FilePublicEvent:
				logger.Debug("slack.FilePublicEvent", zap.Bool("handled", false))
			case *slack.FileSharedEvent:
				logger.Debug("slack.FileSharedEvent", zap.Bool("handled", false))
			case *slack.FileUnsharedEvent:
				logger.Debug("slack.FileUnsharedEvent", zap.Bool("handled", false))

			// Errors
			case *slack.UnmarshallingErrorEvent:
				return ev
			case *slack.OutgoingErrorEvent:
				return ev
			case *slack.AckErrorEvent:
				return ev
			case *slack.RTMError:
				return ev
			case *slack.InvalidAuthEvent:
				logger.Fatal("Slack got InvalidAuthEvent")
			}
		}
	}
}

func GroupInvite(groupID, userID string) error {
	_, _, err := api.InviteUserToGroup(groupID, userID)
	return err
}
