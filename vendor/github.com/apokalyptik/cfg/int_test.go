package cfg

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestInt(t *testing.T) {
	os.Setenv("INT_I1", "1")
	ioutil.WriteFile(".int.json", []byte(`{"i2": 2, "i3": "3"}`), 0644)
	ioutil.WriteFile(".int.yml", []byte("---\ni4: 4\ni5: \"5\""), 0644)
	c := New("int")
	vs := []*int{
		c.Int("i1", 1, ""),
		c.Int("i2", 2, ""),
		c.Int("i3", 3, ""),
		c.Int("i4", 4, ""),
		c.Int("i5", 5, ""),
		c.Int("i6", 6, ""),
	}
	for k, v := range vs {
		var want int = 1 + k
		if *v != want {
			t.Errorf("Expecting %d but got %d", want, *v)
		}
	}
}

func TestInt64(t *testing.T) {
	os.Setenv("INT64_I61", "1")
	ioutil.WriteFile(".int64.json", []byte(`{"i62": 2, "i63": "3"}`), 0644)
	ioutil.WriteFile(".int64.yml", []byte("---\ni64: 4\ni65: \"5\""), 0644)
	c := New("int64")
	vs := []*int64{
		c.Int64("i61", 6, ""),
		c.Int64("i62", 6, ""),
		c.Int64("i63", 6, ""),
		c.Int64("i64", 6, ""),
		c.Int64("i65", 6, ""),
		c.Int64("i66", 6, ""),
	}
	for k, v := range vs {
		var want int64 = 1 + int64(k)
		if *v != want {
			t.Errorf("Expecting %d but got %d", want, *v)
		}
	}
}
