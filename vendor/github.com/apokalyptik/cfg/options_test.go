package cfg

import (
	"fmt"
	"io/ioutil"
	"os/user"
	"testing"
)

func TestOptionsNoFiles(t *testing.T) {
	c := New("none")
	s := c.String("s", "default", "")
	Parse()
	if *s != "default" {
		t.Error("Expected default from no files...")
	}
}

func TestOptionsHomeFiles(t *testing.T) {
	if user, err := user.Current(); err == nil {
		ioutil.WriteFile(fmt.Sprintf("%s/.cfgflagpackagetestdata.json", user.HomeDir), []byte("{\"test\": \"value\"}"), 0644)
		ioutil.WriteFile(fmt.Sprintf("%s/.cfgflagpackagetestdata.yml", user.HomeDir), []byte("{\"test2\": \"value\"}"), 0644)
		c := New("cfgflagpackagetestdata")
		s := c.String("test", "default", "")
		s2 := c.String("test2", "default", "")
		Parse()
		if *s != "value" {
			t.Error("Expected default from home dir files...")
		}
		if *s2 != "value" {
			t.Error("Expected default from home dir files...")
		}
	}
}

func TestOptionsBadFiles(t *testing.T) {
	ioutil.WriteFile(".bad.json", []byte("{\n"), 0644)
	ioutil.WriteFile(".bad.yml", []byte("-blah"), 0644)
	c := New("bad")
	s := c.String("sbad", "default", "")
	Parse()
	if *s != "default" {
		t.Error("Expected default from no files...")
	}
}
