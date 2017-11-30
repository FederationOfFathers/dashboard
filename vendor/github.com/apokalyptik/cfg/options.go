package cfg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"

	"gopkg.in/yaml.v2"
)

// Options is your entry point for the necessary functions for configuring your
// application.  There is some initialization required. Please use New() to
// obtain your very own *Options
type Options struct {
	prefix     string
	yamlData   map[string]interface{}
	jsonData   map[string]interface{}
	mergedData map[string]interface{}
}

func (o *Options) getEnv(name string) string {
	return os.Getenv(o.env(name))
}

func (o *Options) env(name string) string {
	return fmt.Sprintf(
		"%s_%s",
		strings.ToUpper(invalidEnv.ReplaceAllString(o.prefix, "_")),
		strings.ToUpper(invalidEnv.ReplaceAllString(name, "_")),
	)
}

func (o *Options) file(extension string) (string, error) {
	fn := fmt.Sprintf("%s.%s", o.prefix, extension)
	if _, err := os.Stat(fn); err == nil {
		return fn, nil
	}
	fn = fmt.Sprintf(".%s.%s", o.prefix, extension)
	if _, err := os.Stat(fn); err == nil {
		return fn, nil
	}
	if usr, err := user.Current(); err == nil {
		dir := usr.HomeDir
		fn = fmt.Sprintf("%s/.%s.%s", dir, o.prefix, extension)
		if _, err := os.Stat(fn); err == nil {
			return fn, nil
		}
		fn = fmt.Sprintf("%s/%s.%s", dir, o.prefix, extension)
		if _, err := os.Stat(fn); err == nil {
			return fn, nil
		}
	}
	fn = fmt.Sprintf("/etc/%s.%s", o.prefix, extension)
	if _, err := os.Stat(fn); err == nil {
		return fn, nil
	}
	return "", errNotFound
}

func (o *Options) flag(name string) string {
	return invalidEnv.ReplaceAllString(strings.ToLower(o.prefix+"-"+name), "-")
}

func (o *Options) yml() (*os.File, error) {
	fn, err := o.file("yml")
	if err != nil {
		return nil, err
	}
	if Debug {
		log.Println("Found YAML file:", fn)
	}
	return os.Open(fn)
}

func (o *Options) json() (*os.File, error) {
	fn, err := o.file("json")
	if err != nil {
		return nil, err
	}
	if Debug {
		log.Println("Found JSON file:", fn)
	}
	return os.Open(fn)
}

// New returns a new *Options initialized with environment, json, and yaml default configuration
// data for use in accepting command line arguments exactly mirroring a useful subset of the
// flag API for compatibility.
func New(prefix string) *Options {
	o := &Options{
		prefix:     prefix,
		mergedData: map[string]interface{}{},
	}
	if DoJSON {
		if f, err := o.json(); err == nil {
			dec := json.NewDecoder(f)
			dec.UseNumber()
			if err := dec.Decode(&o.jsonData); err != nil {
				o.jsonData = make(map[string]interface{})
				if Debug {
					log.Println("Error decoding JSON:", err.Error())
				}
			}
			f.Close()
		} else {
			if Debug {
				log.Println("No JSON file found")
			}
		}
	} else {
		if Debug {
			log.Println("Skipping JSON")
		}
	}
	if o.jsonData == nil {
		o.jsonData = make(map[string]interface{})
	}
	if DoYAML {
		if f, err := o.yml(); err == nil {
			if buf, err := ioutil.ReadAll(f); err == nil {
				if err := yaml.Unmarshal(buf, &o.yamlData); err != nil {
					o.yamlData = make(map[string]interface{})
					if Debug {
						log.Println("Error parsing YAML:", err.Error())
					}
				}
			} else {
				if Debug {
					log.Println("Error reading YAML:", err.Error())
				}
			}
			f.Close()
		} else {
			if Debug {
				log.Println("no YAML file found")
			}
		}
	} else {
		if Debug {
			log.Println("Skipping YAML")
		}
	}
	if o.yamlData == nil {
		o.yamlData = make(map[string]interface{})
	}
	return o
}
