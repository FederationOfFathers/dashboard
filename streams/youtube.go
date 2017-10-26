package streams

import "go.uber.org/zap"

var ytlog = zap.NewExample().With(zap.String("module", "streams"), zap.String("service", "youtube"))
var YoutubeAPIKey string

func mindYoutube() {
	ytlog.Debug("begin minding")
	for _, stream := range Streams {
		if stream.Youtube == "" {
			continue
		}
		ytlog.Debug("minding", zap.String("key", stream.Youtube))
	}
	ytlog.Debug("end minding")
}
