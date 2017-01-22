package streams

import "github.com/uber-go/zap"

var ytlog = zap.New(zap.NewJSONEncoder(), zap.DebugLevel).With(zap.String("module", "streams"), zap.String("service", "youtube"))
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
