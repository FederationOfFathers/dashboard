package streams

import (
	"fmt"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/messaging"
	"go.uber.org/zap"
	"google.golang.org/api/youtube/v3"
)

var ytlog *zap.Logger
var YoutubeAPIKey string

func mindYoutube() {
	if YouTube == nil {
		Logger.Info("YouTube service not initalized. Not minding Youtube.")
		return
	}
	ytlog = Logger.Named("youtube")
	ytlog.Debug("begin minding")
	for _, stream := range Streams {
		ytlog.Debug("minding", zap.String("key", stream.Youtube))
		if stream.Youtube != "" {
			updateYouTubeStream(stream)
		}
	}
	ytlog.Debug("end minding")
}

func updateYouTubeStream(s *db.Stream) {
	channelID := s.Youtube

	log := Logger.With(zap.Int("stream_id", s.ID), zap.String("channelID", channelID))

	ytSearch := YouTube.Search.List([]string{"snippet"}).
		ChannelId(channelID).Type("video").EventType("live")

	resp, err := ytSearch.Do()
	if err != nil {
		ytlog.With(zap.Error(err), zap.String("channelID", channelID)).Error("unable to check for YouTube stream status")
		return
	}

	if resp.HTTPStatusCode != 200 {
		log.With(zap.Int("status", resp.HTTPStatusCode)).Error("search query failed")
		return
	}

	// the channel is not streaming, ensure we have marked it as not live and done
	if len(resp.Items) == 0 {
		if s.YoutubeStreamID != "" {
			markStreamOffline(s)
			// todo update online message?
		}
		return
	}

	// check if the live stream is one we have already posted, or if its a new stream
	postStreamMessage := false
	var broadcastItem *youtube.SearchResult
	// we have a live stream!
	for _, i := range resp.Items {
		if i.Snippet.LiveBroadcastContent == "live" {
			// update the stream id if it is not the same (also if empty)
			if s.YoutubeStreamID != i.Id.VideoId {
				postStreamMessage = true
				s.YoutubeStreamID = i.Id.VideoId
				// update start stop times
				s.YoutubeStart = time.Now().Unix()
				s.YoutubeStop = time.Now().Unix() - 10
			}
			broadcastItem = i

			// we only need the first active live stream
			break
		}
	}

	if postStreamMessage {
		var c *youtube.Channel
		if resp, err := YouTube.Channels.List([]string{"snippet"}).Id(channelID).Do(); err != nil {
			ytlog.With(
				zap.String("channelID", channelID),
				zap.Int("user", s.MemberID),
				zap.Error(err),
			).Error("unable to get channel info")
		} else {
			if len(resp.Items) > 0 {
				c = resp.Items[0]
			}
		}

		sendYouTubeMessage(broadcastItem, c)
	}

	// save after we send the message. Ideally, we would only do this if the message succeeded
	if err := s.Save(); err != nil {
		ytlog.Error("unable to save YouTube stream data", zap.Any("stream", s), zap.Error(err))
	}

}

func markStreamOffline(s *db.Stream) {
	// offline twitch
	s.TwitchStreamID = ""
	if s.TwitchStop < s.TwitchStart {
		s.TwitchStop = time.Now().Unix()
	}

	// offline youtube
	s.YoutubeStreamID = ""
	if s.YoutubeStop < s.TwitchStart {
		s.YoutubeStop = time.Now().Unix()
	}

	if err := s.Save(); err != nil {
		Logger.With(zap.Error(err)).Error("unable to mark stream as offline")
	}
}

func sendYouTubeMessage(i *youtube.SearchResult, c *youtube.Channel) {

	thumbnailUrl := fmt.Sprintf("%s?%d", i.Snippet.Thumbnails.Medium.Url, time.Now().Unix())
	messaging.SendTwitchStreamMessage(messaging.StreamMessage{
		Platform:         "YouTube",
		PlatformLogo:     "https://slack-imgs.com/?c=1&o1=wi16.he16.si.ip&url=https%3A%2F%2Fwww.youtube.com%2Ffavicon.ico",
		PlatformColor:    "#FF0000",
		PlatformColorInt: 16711680,
		Username:         i.Snippet.ChannelTitle,
		UserLogo:         c.Snippet.Thumbnails.Default.Url,
		URL:              fmt.Sprintf("https://youtube.com/v/%s", i.Id.VideoId),
		Description:      i.Snippet.Title,
		Timestamp:        time.Now().Format("01/02/2006 15:04 MST"),
		ThumbnailURL:     thumbnailUrl,
	})
}
