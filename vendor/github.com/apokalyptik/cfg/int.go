package cfg

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
)

func (o *Options) defaultInt64(name string, value int64) int64 {
	if v := o.getEnv(name); v != "" {
		if i, err := strconv.ParseInt(v, 0, 64); err == nil {
			value = i
		}
	}
	if v, ok := o.jsonData[name]; ok {
		switch t := v.(type) {
		case string:
			if i, err := strconv.ParseInt(t, 0, 64); err == nil {
				value = i
			}
		case json.Number:
			if val, ok := v.(json.Number); ok {
				if i, err := val.Int64(); err == nil {
					value = int64(i)
				}
			}
		}
	}
	if v, ok := o.yamlData[name]; ok {
		switch t := v.(type) {
		case string:
			if i, err := strconv.ParseInt(t, 0, 64); err == nil {
				value = i
			}
		case int, uint, int32, uint32, int64, uint64:
			if i, err := strconv.ParseInt(fmt.Sprintf("%d", t), 0, 64); err == nil {
				value = i
			}
		}
	}
	return value
}

// Int64 works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Int64(name string, value int64, usage string) *int64 {
	return flag.Int64(o.flag(name), o.defaultInt64(name, value), usage)
}

// Int64Var works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Int64Var(p *int64, name string, value int64, usage string) {
	flag.Int64Var(p, o.flag(name), o.defaultInt64(name, value), usage)
}

func (o *Options) defaultInt(name string, value int) int {
	return int(o.defaultInt64(name, int64(value)))
}

// Int works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Int(name string, value int, usage string) *int {
	return flag.Int(o.flag(name), o.defaultInt(name, value), usage)
}

// IntVar works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) IntVar(p *int, name string, value int, usage string) {
	flag.IntVar(p, o.flag(name), o.defaultInt(name, value), usage)
}
