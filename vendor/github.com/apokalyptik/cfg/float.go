package cfg

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

func (o *Options) defaultFloat(name string, value float64) float64 {
	if v := os.Getenv(o.env(name)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			value = f
		}
	}
	if v, ok := o.jsonData[name]; ok {
		switch t := v.(type) {
		case string:
			if f, err := strconv.ParseFloat(t, 64); err == nil {
				value = f
			}
		case json.Number:
			if val, err := t.Float64(); err == nil {
				value = val
			}
		}
	}
	if v, ok := o.yamlData[name]; ok {
		switch t := v.(type) {
		case string:
			if f, err := strconv.ParseFloat(t, 64); err == nil {
				value = f
			}
		case float64:
			value = t
		case int, uint, int32, uint32, int64, uint64:
			if val, err := strconv.ParseFloat(fmt.Sprintf("%d", t), 64); err == nil {
				value = val
			}
		}
	}
	return value
}

// Float64 works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Float64(name string, value float64, usage string) *float64 {
	return flag.Float64(o.flag(name), o.defaultFloat(name, value), usage)
}

// Float64Var works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Float64Var(p *float64, name string, value float64, usage string) {
	flag.Float64Var(p, o.flag(name), o.defaultFloat(name, value), usage)
}
