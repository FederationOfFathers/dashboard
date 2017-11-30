package cfg

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
)

func (o *Options) defaultUint64(name string, value uint64) uint64 {
	if v := o.getEnv(name); v != "" {
		if i, err := strconv.ParseInt(v, 0, 64); err == nil {
			if i >= 0 {
				value = uint64(i)
			}
		}
	}
	if v, ok := o.jsonData[name]; ok {
		switch t := v.(type) {
		case string:
			if i, err := strconv.ParseInt(t, 0, 64); err == nil {
				if i >= 0 {
					value = uint64(i)
				}
			}
		case json.Number:
			if val, ok := v.(json.Number); ok {
				if i, err := val.Int64(); err == nil {
					if i >= 0 {
						value = uint64(i)
					}
				}
			}
		}
	}
	if v, ok := o.yamlData[name]; ok {
		switch t := v.(type) {
		case string:
			if i, err := strconv.ParseInt(t, 0, 64); err == nil {
				if i >= 0 {
					value = uint64(i)
				}
			}
		case int, uint, int32, uint32, int64, uint64:
			if i, err := strconv.ParseInt(fmt.Sprintf("%d", t), 0, 64); err == nil {
				if i >= 0 {
					value = uint64(i)
				}
			}
		}
	}
	return value
}

// Uint64 works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Uint64(name string, value uint64, usage string) *uint64 {
	return flag.Uint64(o.flag(name), o.defaultUint64(name, value), usage)
}

// Uint64Var works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Uint64Var(p *uint64, name string, value uint64, usage string) {
	flag.Uint64Var(p, o.flag(name), o.defaultUint64(name, value), usage)
}

func (o *Options) defaultUint(name string, value uint) uint {
	return uint(o.defaultUint64(name, uint64(value)))
}

// Uint works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) Uint(name string, value uint, usage string) *uint {
	return flag.Uint(o.flag(name), o.defaultUint(name, value), usage)
}

// UintVar works mostly like the flag package equivalent except that it will pull in defaults from the environment, and configuratin files
func (o *Options) UintVar(p *uint, name string, value uint, usage string) {
	flag.UintVar(p, o.flag(name), o.defaultUint(name, value), usage)
}
