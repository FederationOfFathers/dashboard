package cfg

import (
	"flag"
	"os"
	"strings"
)

func (o *Options) defaultBool(name string, value bool) bool {
	if v := strings.ToLower(os.Getenv(o.env(name))); v != "" {
		switch v[:1] {
		case "1":
			value = true
		case "t":
			if v[:4] == "true" {
				value = true
			}
		case "y":
			if v == "y" || v == "yes" {
				value = true
			}
		}
	}
	if v, ok := o.jsonData[name]; ok {
		if val, ok := v.(bool); ok {
			value = val
		}
	}
	if v, ok := o.yamlData[name]; ok {
		if val, ok := v.(bool); ok {
			value = val
		}
	}
	return value
}

// Bool works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Bool(name string, value bool, usage string) *bool {
	return flag.Bool(o.flag(name), o.defaultBool(name, value), usage)
}

// BoolVar works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) BoolVar(p *bool, name string, value bool, usage string) {
	flag.BoolVar(p, o.flag(name), o.defaultBool(name, value), usage)
}
