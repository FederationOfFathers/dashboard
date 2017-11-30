/*
Package cfgflag is an easy way to provide configuration file and environmental
defaults to the flag package from the standard library.

The reason for the creation of this package was to make a simple, easy, and
standard library based way to configure applications and libs. There exist many
very very good configuration libraries out there for Go, but the ones that
I have found are very heavy and usually geared towards a very specific kind
of use case.  They rely a lot on reflection, and configuring entire structs
at once.

This is meant to be a drop in replacement for the most common use
cases of the flag package (indeed it uses the flag package to do most of the
work) where you would like to be able to confugure an application via
files or environment variables.  An example of why one might want to do this
is when you're using flags during development but in production you do not
wish to have sensitive data like api keys and passwords listed in the
process list.

Since it also presents you with a namespace. It's very easy to adopt in
libraries where specific configuration would be useful.  An example of
which might be a library which requires an API key but that might be shared
between many processed or commands, but it is undesireable to configure
the library manually each time you wish to use it.  In this case dropping
a json or yaml configuration into your home dir, or users environment
would configure the library for all processes and commands at once.

	c := cfgflag.New("test")
	myvar := c.Int("myint", 100, "my integer flag")
	cfgflag.Parse

Given the preceeding code you can expect, in order from lowest priority to highest:

Standard library flag defaults for -test-myint (prefix + flag)
	  -test-myint int
			my integer flag (default 100)
At this point *myvar would be 100 if no further configuration is found or specified.

Environment variable based defaults via
	TEST_MYINT=98
You'll notice that it's {prefix}_{flag} all uppercased. At this point *myvar would be 98
if no further configuration is found or specified.

JSON config file support for
	{ "myint": 97 }
You'll notice that there is no prefix inside the file since the filename counts as the
variables namespace. JSON is read from exactly one the following files in order from
lowest priority to highest:
	/etc/test.json unless we find
	~/.test.json unless we find
	~/test.json unless we find
	./.test.json unless we find
	./test.json
At this point *myvar would be 97 if no further configuration is found or specified.

YAML config file support for
	---
	myint: 96
You'll notice that there is no prefix inside the file since the filename counts as the
variables namespace. YAML is read from exactly one the following files in order from
lowest priority to highest:
	/etc/test.yml unless we find
	~/.test.yml unless we find
	~/test.yml unless we find
	./.test.yml unless we find
	./test.yml
At this point *myvar would be 96 if no further configuration is found or specified.

Command line speficied flags take precedence over YAML, JSON, ENV, and flag defaults
	-test-myint=95
At this point *myvar would be 95.
*/
package cfg

import (
	"flag"
	"fmt"
	"regexp"
)

var (
	// Debug tells us whether to log debugging messages
	Debug = false
	// DoYAML tells us whether to look for YAML configuration files
	DoYAML = true
	// DoJSON tells us whether to look for JSON configuration files
	DoJSON      = true
	invalidEnv  = regexp.MustCompile("[^0-9a-zA-Z]")
	errNotFound = fmt.Errorf("file not found")
)

// Parse is a convenience wrapper for flag.Parse
func Parse() {
	flag.Parse()
}
