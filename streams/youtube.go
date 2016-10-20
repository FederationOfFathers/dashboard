package streams

import "github.com/uber-go/zap"

var ytlog = zap.New(zap.NewJSONEncoder()).With(zap.String("module", "streams"), zap.String("service", "twitch"))
var YoutubeAPIKey string

func (s *Stream) updateYoutube() {
}

func mindYoutube() {
	ytlog.Debug("begin minding")
	for key, stream := range Streams {
		if stream.Kind != "youtube" {
			continue
		}
		ytlog.Debug("minding", zap.String("key", key))
		stream.update()
	}
	ytlog.Debug("end minding")
}
