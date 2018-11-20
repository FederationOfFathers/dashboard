package metrics

import (
	"github.com/FederationOfFathers/dashboard/environment"
	"github.com/rollbar/rollbar-go"
	"go.uber.org/zap/zapcore"
)

// RollbarConfig config for rollbar integration
type RollbarConfig struct {
	Token       string `yaml:"token"`
	Environment string `yaml:"environment"`
}

type levelEnabler struct {
}

// Init starts up rollbar. Without this, there are no rollbar metrics.
func (r *RollbarConfig) Init() {

	rollbar.SetToken(r.Token)

	if r.Environment != "" {
		rollbar.SetEnvironment(r.Environment)
	} else if environment.IsProd {
		rollbar.SetEnvironment("production")
	} else if environment.IsDev {
		rollbar.SetEnvironment("development")
	} else if environment.IsLocal {
		rollbar.SetEnvironment("localhost")
	}

	rollbar.Wait()
}

// RollbarZapCore is a custom core to send logs to Rollbar. Add the core using zapcore.NewTee
type RollbarZapCore struct {
	levelEnabler
	coreFields map[string]interface{}
}

// NewRollbarCore creates a new RollbarZapCore
func NewRollbarCore() *RollbarZapCore {
	return &RollbarZapCore{
		coreFields: make(map[string]interface{}),
	}
}

func (le *levelEnabler) Enabled(l zapcore.Level) bool {
	return l >= zapcore.WarnLevel
}

// With provides structure
func (c *RollbarZapCore) With(fields []zapcore.Field) zapcore.Core {

	fieldMap := fieldsToMap(fields)

	for k, v := range fieldMap {
		c.coreFields[k] = v
	}

	return c
}

// Check determines if this should be sent to roll bar based on LevelEnabler
func (c *RollbarZapCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.levelEnabler.Enabled(entry.Level) {
		return checkedEntry.AddCore(entry, c)
	}
	return checkedEntry
}

func (c *RollbarZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {

	fieldMap := fieldsToMap(fields)
	for k, v := range fieldMap {
		c.coreFields[k] = v
	}

	c.coreFields["logger"] = entry.LoggerName
	c.coreFields["file"] = entry.Caller.TrimmedPath()

	switch entry.Level {
	case zapcore.ErrorLevel:
		rollbar.Error(entry.Message, c.coreFields)
	case zapcore.WarnLevel:
		rollbar.Warning(entry.Message, c.coreFields)
	case zapcore.DPanicLevel:
		rollbar.Critical(entry.Message, c.coreFields)
	}
	return nil
}

// Sync flushes
func (c *RollbarZapCore) Sync() error {
	rollbar.Wait()
	return nil
}

func fieldsToMap(fields []zapcore.Field) map[string]interface{} {
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}

	m := make(map[string]interface{})
	for k, v := range enc.Fields {
		m[k] = v
	}
	return m
}
