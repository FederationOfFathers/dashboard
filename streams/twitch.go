package streams

import (
	"fmt"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/messaging"
	"github.com/nicklaw5/helix"
	"go.uber.org/zap"
)

var twlog *zap.Logger
var TwitchOAuthKey string

var twitchClient *helix.Client

func Twitch(clientID string) error {
	var err error
	twitchClient, err = helix.NewClient(&helix.Options{
		ClientID: clientID,
	})
	return err

}

func MustTwitch(oauth string) {
	if err := Twitch(oauth); err != nil {
		panic(err)
	}
}

type twitchStream helix.Stream

func mindTwitch() {
	twlog = Logger.Named("twitch")
	twlog.Debug("begin minding")
	var streamsCount int
	for _, stream := range Streams {
		if stream.Twitch == "" {
			twlog.Debug("not a twitch stream", zap.Int("id", stream.ID), zap.Int("member_id", stream.MemberID))
			continue
		}
		twlog.Debug("minding twitch stream", zap.String("Twitch id", stream.Twitch))
		updateTwitch(stream)
		streamsCount++
	}
	twlog.Info("twitch streams updated", zap.Int("numStreams", streamsCount))
	twlog.Debug("end minding")
}

func updateTwitch(s *db.Stream) {
	var client = twitchClient
	var foundStream = false

	// retrieve the stream by username
	res, err := client.GetStreams(&helix.StreamsParams{
		UserLogins: []string{s.Twitch},
	})
	if err != nil {
		if err.Error() != "json: cannot unmarshal number into Go value of type string" {
			twlog.Error("error fetching twitch stream", zap.String("key", s.Twitch), zap.Error(err))
		}
		return
	}

	switch len(res.Data.Streams) {
	case 1:
		foundStream = true
	case 0:
		twlog.Info("No active streams", zap.String("key", s.Twitch))
	default:
		twlog.Error("Too many active streams", zap.String("key", s.Twitch))
	}

	if !foundStream {
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
		return
	}

	stream := res.Data.Streams[0]

	if stream.ID == "" {
		twlog.Error("Invalid stream ID", zap.String("key", s.Twitch))
		return
	}

	var isRecent bool = time.Now().Unix()-s.TwitchStart <= 1800
	streamID := fmt.Sprintf("%s", stream.ID)
	postStreamMessage := true
	if streamID == s.TwitchStreamID && s.TwitchGame == stream.GameID {
		twlog.Debug("still streaming...", zap.String("twitch_user", s.Twitch), zap.String("game_id", stream.GameID))
		return
	} else if isRecent && s.TwitchGame == stream.GameID {
		twlog.Debug("new ID, but still streaming...", zap.String("twitch_user", s.Twitch), zap.String("game_id", stream.GameID))
		postStreamMessage = false
	}

	s.TwitchStreamID = streamID
	s.TwitchStart = time.Now().Unix()
	if s.TwitchStop > s.TwitchStart {
		s.TwitchStop = s.TwitchStart - 1
	}

	var game helix.Game
	gamesResponse, gerr := client.GetGames(&helix.GamesParams{
		IDs: []string{stream.GameID},
	})
	if gerr != nil {
		twlog.Error("could not get game data", zap.Error(err), zap.String("gameID", stream.GameID), zap.String("twitchUser", stream.UserName))
	} else {
		game = gamesResponse.Data.Games[0]

	}

	if postStreamMessage {
		twlog.Info("posting twistream message")
		sendTwitchMessage(stream, game)
	}

	s.TwitchGame = stream.GameID
	if err := s.Save(); err != nil {
		twlog.Error("unable to save stream data", zap.Any("stream", s), zap.Error(err))
	}
}

func sendTwitchMessage(stream helix.Stream, game helix.Game) {

	messaging.SendTwitchStreamMessage(messaging.StreamMessage{
		Platform:         "Twitch",
		PlatformLogo:     "https://slack-imgs.com/?c=1&o1=wi16.he16.si.ip&url=https%3A%2F%2Fwww.twitch.tv%2Ffavicon.ico",
		PlatformColor:    "#6441A4",
		PlatformColorInt: 6570404,
		Username:         stream.UserName,
		UserLogo:         stream.ThumbnailURL,
		URL:              fmt.Sprintf("https://twitch.tv/%s", stream.UserName),
		Game:             game.Name,
		Description:      stream.Title,
		Timestamp:        time.Now().Format("01/02/2006 15:04 MST"),
	})

}
