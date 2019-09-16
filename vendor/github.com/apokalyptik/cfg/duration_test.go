package cfg

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestDuration(t *testing.T) {
	os.Setenv("DUR_D1", "1m0s")
	ioutil.WriteFile(".dur.json", []byte(`{"d2": "1m1s"}`), 0644)
	ioutil.WriteFile(".dur.yml", []byte("---\nd3: 1m2s"), 0644)
	c := New("dur")
	dDur := time.Minute + (3 * time.Second)
	vs := []*time.Duration{
		c.Duration("d1", dDur, ""),
		c.Duration("d2", dDur, ""),
		c.Duration("d3", dDur, ""),
		c.Duration("d4", dDur, ""),
	}
	Parse()
	for k, v := range vs {
		want := fmt.Sprintf("1m%ds", k)
		if v.String() != want {
			t.Errorf("Expected %s but got %s", want, v.String())
		}
	}
}
