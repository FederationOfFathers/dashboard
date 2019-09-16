package cfg

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestUint(t *testing.T) {
	os.Setenv("UINT_UI1", "1")
	ioutil.WriteFile(".uint.json", []byte(`{"ui2": 2, "ui3": "3"}`), 0644)
	ioutil.WriteFile(".uint.yml", []byte("---\nui4: 4\nui5: \"5\""), 0644)
	c := New("uint")
	vs := []*uint{
		c.Uint("ui1", 1, ""),
		c.Uint("ui2", 2, ""),
		c.Uint("ui3", 3, ""),
		c.Uint("ui4", 4, ""),
		c.Uint("ui5", 5, ""),
		c.Uint("ui6", 6, ""),
	}
	for k, v := range vs {
		var want uint = 1 + uint(k)
		if *v != want {
			t.Errorf("Expecting %d but got %d", want, *v)
		}
	}
}

func TestUint64(t *testing.T) {
	os.Setenv("UINT64_UI61", "1")
	ioutil.WriteFile(".uint64.json", []byte(`{"ui62": 2, "ui63": "3"}`), 0644)
	ioutil.WriteFile(".uint64.yml", []byte("---\nui64: 4\nui65: \"5\""), 0644)
	c := New("uint64")
	vs := []*uint64{
		c.Uint64("ui61", 6, ""),
		c.Uint64("ui62", 6, ""),
		c.Uint64("ui63", 6, ""),
		c.Uint64("ui64", 6, ""),
		c.Uint64("ui65", 6, ""),
		c.Uint64("ui66", 6, ""),
	}
	for k, v := range vs {
		var want uint64 = 1 + uint64(k)
		if *v != want {
			t.Errorf("Expecting %d but got %d", want, *v)
		}
	}
}
