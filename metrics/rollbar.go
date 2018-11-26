package metrics

import (
	"github.com/FederationOfFathers/dashboard/environment"
	"github.com/rollbar/rollbar-go"
)

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
	} else if environment.IsProd {
		rollbar.SetEnvironment("production")
	} else if environment.IsDev {
		rollbar.SetEnvironment("development")
	} else if environment.IsLocal {
		rollbar.SetEnvironment("localhost")
	}

	rollbar.Wait()
}
