package streams

import "go.uber.org/zap"

var ytlog *zap.Logger
var YoutubeAPIKey string

func mindYoutube() {
	ytlog = Logger.With(zap.String("service", "youtube"))
	ytlog.Debug("begin minding")
	for _, stream := range Streams {
		if stream.Youtube == "" {
			continue
		}
		ytlog.Debug("minding", zap.String("key", stream.Youtube))
	}
	ytlog.Debug("end minding")
}
