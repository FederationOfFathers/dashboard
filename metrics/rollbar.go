package metrics

import (
	"os"

	"github.com/rollbar/rollbar-go"
	"go.uber.org/zap/zapcore"
)

const prodServicePath = "/var/lib/dashboard/prod"

// RollbarConfig config for rollbar integration
type RollbarConfig struct {
	Token       string `yaml:"token"`
	Environment string `yaml:"environment"`
}

// Init starts up rollbar. Without this, there are no rollbar metrics.
func (r *RollbarConfig) Init() {

	rollbar.SetToken(r.Token)

	if r.Environment != "" {
		rollbar.SetEnvironment(r.Environment)
	} else if _, err := os.Stat(prodServicePath); !os.IsNotExist(err) { // need a better environment flag
		rollbar.SetEnvironment("production")
	}

	rollbar.Wait()
}

// LoggerHook zapcore Hook for Rollbar. Outputs only error and warn messages
func (r *RollbarConfig) LoggerHook(entry zapcore.Entry) error {

	// run only if rollbar is configured
	if r.Token == "" {
		return nil
	}
	go func(e zapcore.Entry) {

		data := map[string]interface{}{
			"logger": e.LoggerName,
			"file":   e.Caller.TrimmedPath(),
		}
		switch e.Level {
		case zapcore.ErrorLevel:
			rollbar.Error(e.Message, data)
		case zapcore.WarnLevel:
			rollbar.Warning(e.Message, data)
		case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
			rollbar.Critical(e.Message, data)
		}

	}(entry)

	return nil

}
