package cfg

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestBoolOptions(t *testing.T) {
	ioutil.WriteFile(".bool.json", []byte(`{
		"jt1": true,
		"jt2": false
		}`), 0644)
	ioutil.WriteFile(".bool.yml", []byte("---\nyt1: true\nyt2: false\n"), 0644)
	os.Setenv("BOOL_ET1", "Yes")
	os.Setenv("BOOL_ET2", "y")
	os.Setenv("BOOL_ET3", "1")
	os.Setenv("BOOL_ET4", "true")
	c := New("bool")
	et1 := c.Bool("et1", false, "")
	et2 := c.Bool("et2", false, "")
	et3 := c.Bool("et3", false, "")
	et4 := c.Bool("et4", false, "")
	et5 := c.Bool("et5", false, "")
	jt1 := c.Bool("jt1", false, "")
	jt2 := c.Bool("jt2", true, "")
	yt1 := c.Bool("yt1", false, "")
	yt2 := c.Bool("yt2", true, "")
	Parse()
	if *et1 == false {
		t.Errorf("expected BOOL_ET1='Yes' to yeild true")
	}
	if *et2 == false {
		t.Errorf("expected BOOL_ET2='y' to yeild true")
	}
	if *et3 == false {
		t.Errorf("expected BOOL_ET3='1' to yeild true")
	}
	if *et4 == false {
		t.Errorf("expected BOOL_ET4='true' to work")
	}
	if *et5 == true {
		t.Errorf("expected default false to work")
	}
	if *jt1 == false {
		t.Errorf("expected jt1: true to make true")
	}
	if *jt2 == true {
		t.Errorf("expected jt2: false to make false")
	}
	if *yt1 == false {
		t.Errorf("expected yt1: true to make true")
	}
	if *yt2 == true {
		t.Errorf("expected yt2: false to make false")
	}

}
