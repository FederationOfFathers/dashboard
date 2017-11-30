package cfg

import (
	"flag"
	"os"
	"time"
)

func (o *Options) defaultDuration(name string, value time.Duration) time.Duration {
	if v := os.Getenv(o.env(name)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			value = d
		}
	}
	if v, ok := o.jsonData[name]; ok {
		if val, ok := v.(string); ok {
			if d, err := time.ParseDuration(val); err == nil {
				value = d
			}
		}
	}
	if v, ok := o.yamlData[name]; ok {
		if val, ok := v.(string); ok {
			if d, err := time.ParseDuration(val); err == nil {
				value = d
			}
		}
	}
	return value
}

// Duration works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Duration(name string, value time.Duration, usage string) *time.Duration {
	return flag.Duration(o.flag(name), o.defaultDuration(name, value), usage)
}

// DurationVar works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	flag.DurationVar(p, o.flag(name), o.defaultDuration(name, value), usage)
}
