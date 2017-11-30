package cfg

import "flag"

func (o *Options) defaultString(name, value string) string {
	if v := o.getEnv(name); v != "" {
		value = v
	}
	if v, ok := o.jsonData[name]; ok {
		if val, ok := v.(string); ok {
			value = val
		}
	}
	if v, ok := o.yamlData[name]; ok {
		switch t := v.(type) {
		case string:
			value = t
		}
	}
	return value
}

// String works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) String(name string, value string, usage string) *string {
	return flag.String(o.flag(name), o.defaultString(name, value), usage)
}

// StringVar works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) StringVar(p *string, name string, value string, usage string) {
	flag.StringVar(p, o.flag(name), o.defaultString(name, value), usage)
}
