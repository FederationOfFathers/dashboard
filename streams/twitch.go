package streams

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/messaging"
	"github.com/honeycombio/beeline-go"
	"github.com/nicklaw5/helix"
	"go.uber.org/zap"
)

var twlog *zap.Logger
var TwitchOAuthKey string

var twitchClient *helix.Client

type token struct {
	token   string
	expires time.Time
}

var twitchToken token

func Twitch(clientID string, clientSecret string) error {
	var err error
	twitchClient, err = helix.NewClient(&helix.Options{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	return err

}

func ensureClientAccess() error {
	if twitchToken.expires.Unix() <= time.Now().Unix() {
		twlog.Info("refreshing expired Twitch token")
		token, err := twitchClient.RequestAppAccessToken([]string{})
		if err != nil {
			return err
		}
		twitchToken.token = token.Data.AccessToken
		twitchToken.expires = time.Now().Add(time.Second * time.Duration(token.Data.ExpiresIn))
		twitchClient.SetAppAccessToken(token.Data.AccessToken)
	}
	return nil
}

type twitchStream helix.Stream

func mindTwitch() {
	ctx, span := beeline.StartSpan(context.Background(), "mindTwitch")
	defer span.Send()
	twlog = Logger.Named("twitch")
	twlog.Debug("begin minding")

	updateTwitch(ctx, Streams)

	twlog.Debug("twitch streams updated") //, zap.Int("numStreams", streamsCount))
	twlog.Debug("end minding")
}

func updateTwitch(ctx context.Context, streams []*db.Stream) {
	var client = twitchClient
	if err := ensureClientAccess(); err != nil {
		twlog.Error("unable to verify twitch client app token", zap.Error(err))
		return
	}

	var userLogins []string
	var indexedStreams = make(map[string]*db.Stream, len(streams))
	for _, s := range streams {
		if s.Twitch != "" {
			userLogins = append(userLogins, s.Twitch)
			indexedStreams[strings.ToLower(s.Twitch)] = s
		}
	}

	// if there are no Twitch users, then nothing left to do
	if len(userLogins) == 0 {
		return
	}

	// get users info and index to a map. continue on failures
	var indexedUsers = make(map[string]helix.User, len(userLogins))
	if usersRes, err := client.GetUsers(&helix.UsersParams{Logins: userLogins}); err != nil {
		twlog.Error("get users call failed", zap.Error(err), zap.Strings("users", userLogins))
	} else {
		for _, u := range usersRes.Data.Users {
			indexedUsers[u.ID] = u
		}
	}

	// retrieve the streams by username
	res, err := client.GetStreams(&helix.StreamsParams{
		UserLogins: userLogins,
	})
	if err != nil {
		if err.Error() != "json: cannot unmarshal number into Go value of type string" {
			twlog.Error("error fetching twitch stream", zap.Strings("userLogins", userLogins), zap.Error(err))
		}
		return
	}

	if res.ErrorStatus != 0 {
		twlog.Error("twitch response returned an error", zap.Int("errorStatus", res.ErrorStatus), zap.String("error", fmt.Sprintf("%s: %s", res.Error, res.ErrorMessage)))
		return
	}

	for _, stream := range res.Data.Streams {
		// get the db value and remove it from the indexed map
		s := indexedStreams[strings.ToLower(stream.UserName)]
		delete(indexedStreams, strings.ToLower(stream.UserName))

		var isRecent bool = time.Now().Unix()-s.TwitchStart <= 1800
		streamID := fmt.Sprintf("%s", stream.ID)
		postStreamMessage := true

		if streamID == s.TwitchStreamID {
			// if the stream has the same ID and the same game, then leave it as is
			twlog.Debug("still streaming...", zap.String("twitch_user", s.Twitch), zap.String("game_id", stream.GameID))
			continue
		} else if isRecent {
			// if the game ID hasn't changed update, but don't sent a message
			twlog.Debug("new ID, but still streaming...", zap.String("twitch_user", s.Twitch), zap.String("game_id", stream.GameID))
			postStreamMessage = false
		}

		s.TwitchStreamID = streamID
		s.TwitchStart = time.Now().Unix()
		if s.TwitchStop > s.TwitchStart {
			s.TwitchStop = s.TwitchStart - 1
		}

		if postStreamMessage {
			twlog.Info("posting twistream message")
			var u helix.User
			if user, ok := indexedUsers[stream.UserID]; ok {
				u = user
			}
			sendTwitchMessage(stream, u)
		}

		if err := s.Save(); err != nil {
			twlog.Error("unable to save Twitch stream data", zap.Any("stream", s), zap.Error(err))
		}

	}

	// update remaining streams as not streaming
	for _, s := range indexedStreams {
		var save bool
		// clear out existing stream ID
		if s.TwitchStreamID != "" {
			s.TwitchStreamID = ""
			save = true
		}
		if s.TwitchStop < s.TwitchStart {
			s.TwitchStop = time.Now().Unix()
			save = true
		}
		if s.TwitchStop < s.TwitchStart {
			s.TwitchStop = s.TwitchStart + 1
			save = true
		}
		if save {
			if err := s.Save(); err != nil {
				twlog.Error("could not save twitch stream", zap.Error(err))
			}
		}
	}

}

func sendTwitchMessage(stream helix.Stream, user helix.User) {

	var userLogo string
	if user.ID != "" {
		userLogo = user.ProfileImageURL
	}

	thumbnailUrl := strings.Replace(stream.ThumbnailURL, "{width}", "320", 1)
	thumbnailUrl = strings.Replace(thumbnailUrl, "{height}", "180", 1)
	thumbnailUrl = fmt.Sprintf("%s?%d", thumbnailUrl, time.Now().Unix())

	messaging.SendTwitchStreamMessage(messaging.StreamMessage{
		Platform:         "Twitch",
		PlatformLogo:     "https://slack-imgs.com/?c=1&o1=wi16.he16.si.ip&url=https%3A%2F%2Fwww.twitch.tv%2Ffavicon.ico",
		PlatformColor:    "#6441A4",
		PlatformColorInt: 6570404,
		Username:         stream.UserName,
		UserLogo:         userLogo,
		URL:              fmt.Sprintf("https://twitch.tv/%s", stream.UserName),
		Description:      stream.Title,
		Timestamp:        time.Now().Format("01/02/2006 15:04 MST"),
		ThumbnailURL:     thumbnailUrl,
	})

}
