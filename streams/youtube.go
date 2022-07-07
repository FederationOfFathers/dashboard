package streams

import (
	"fmt"
	"net/url"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/messaging"
	"github.com/gocolly/colly/v2"
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

	// indexed by channelID
	indexedStreams := make(map[string]*db.Stream)
	var streamsToCheck []string
	for _, stream := range Streams {

		if stream.Youtube != "" {
			ytlog.With(
				zap.String("channelID", stream.Youtube),
			).Debug("minding stream")
			indexedStreams[stream.Youtube] = stream

			// get the 'live' video id by scraping the '/live' page for the channel
			vidId := getYTVideoIDForStream(stream)
			if vidId != "" {
				streamsToCheck = append(streamsToCheck, vidId)
			} else if stream.YoutubeStreamID != "" {
				// if the vidID was empty and stream ID was not, we need to mark it as offline
				markStreamOffline(stream)
			}

		}
	}

	if len(streamsToCheck) == 0 {
		ytlog.Debug("no YouTube streams found")
		return
	}

	vids := getYTStreamStatuses(streamsToCheck)
	if vids == nil {
		return
	}

	var updatedStreams []*db.Stream
	var liveChannelIDs []string
	indexedVids := make(map[string]*youtube.Video)
	for _, v := range vids {
		channelID := v.Snippet.ChannelId

		s, found := indexedStreams[channelID]
		if !found {
			ytlog.With(zap.String("channelID", channelID)).Error("impossible channel id without a stream")
			continue
		}

		if v.Snippet.LiveBroadcastContent == "live" {
			// online and needs updating
			if s.YoutubeStreamID != v.Id {

				s.YoutubeStreamID = v.Id
				s.YoutubeStart = time.Now().Unix()
				s.YoutubeStop = 0

				liveChannelIDs = append(liveChannelIDs, channelID)
				updatedStreams = append(updatedStreams, s)
				indexedVids[channelID] = v
			}

		} else if s.YoutubeStreamID != "" {
			//offline
			markStreamOffline(s)
		}
	}

	channels := getYTChannelInfo(liveChannelIDs)
	for _, c := range channels {
		vid, found := indexedVids[c.Id]
		if !found {
			ytlog.With(zap.String("channelID", c.Id)).Error("impossible channel id without a video")
			continue
		}

		sendYouTubeMessage(vid, c)
	}

	// save all update streams
	for _, s := range updatedStreams {
		if err := s.Save(); err != nil {
			ytlog.With(zap.String("yt_channel", s.Youtube), zap.Error(err)).Error("unable to save stream data")
		}
	}

	ytlog.Debug("end minding")
}

func getYTChannelInfo(ids []string) []*youtube.Channel {
	if len(ids) == 0 {
		return []*youtube.Channel{}
	}
	resp, err := YouTube.Channels.List([]string{"snippet"}).Id(ids...).Do()
	if err != nil {
		ytlog.With(zap.Strings("ids", ids), zap.Error(err)).Error("unable to get channel info")
		return []*youtube.Channel{}
	}

	return resp.Items
}
func getYTVideoIDForStream(s *db.Stream) string {
	vidUrl := getCanonicalURL(fmt.Sprintf("https://www.youtube.com/channel/%s/live", s.Youtube))
	u, err := url.Parse(vidUrl)
	if err != nil {
		ytlog.With(
			zap.String("url", vidUrl),
			zap.String("yt_channel", s.Youtube),
			zap.Error(err),
		).Error("unable to parse url")
	}

	return u.Query().Get("v")
}

func getCanonicalURL(url string) string {
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

func getYTStreamStatuses(ids []string) []*youtube.Video {

	resp, err := YouTube.Videos.List([]string{"snippet"}).Id(ids...).Do()
	if err != nil {
		ytlog.With(zap.Strings("ids", ids), zap.Error(err)).Error("unable to check YouTube streams")
		return nil
	}

	return resp.Items

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

func sendYouTubeMessage(i *youtube.Video, c *youtube.Channel) {

	thumbnailUrl := fmt.Sprintf("%s?%d", i.Snippet.Thumbnails.Medium.Url, time.Now().Unix())
	messaging.SendTwitchStreamMessage(messaging.StreamMessage{
		Platform:         "YouTube",
		PlatformLogo:     "https://slack-imgs.com/?c=1&o1=wi16.he16.si.ip&url=https%3A%2F%2Fwww.youtube.com%2Ffavicon.ico",
		PlatformColor:    "#FF0000",
		PlatformColorInt: 16711680,
		Username:         i.Snippet.ChannelTitle,
		UserLogo:         c.Snippet.Thumbnails.Default.Url,
		URL:              fmt.Sprintf("https://youtube.com/v/%s", i.Id),
		Description:      i.Snippet.Title,
		Timestamp:        time.Now().Format("01/02/2006 15:04 MST"),
		ThumbnailURL:     thumbnailUrl,
	})
}
