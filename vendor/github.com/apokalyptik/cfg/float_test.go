package cfg

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestFloat(t *testing.T) {
	os.Setenv("FLOAT_F1", "0.1")
	ioutil.WriteFile(".float.json", []byte(`{"f2": 0.2, "f3": "0.3", "f7": 7}`), 0644)
	ioutil.WriteFile(".float.yml", []byte("---\nf4: 0.4\nf5: \"0.5\"\nf8: 8"), 0644)
	c := New("float")
	vs := []*float64{
		c.Float64("f1", 0.6, ""),
		c.Float64("f2", 0.6, ""),
		c.Float64("f3", 0.6, ""),
		c.Float64("f4", 0.6, ""),
		c.Float64("f5", 0.6, ""),
		c.Float64("f6", 0.6, ""),
	}
	f7 := c.Float64("f7", 0.6, "")
	f8 := c.Float64("f8", 0.6, "")
	for k, v := range vs {
		var want float64 = 0.1 + 0.1*float64(k)
		if fmt.Sprintf("%f", *v) != fmt.Sprintf("%f", want) {
			t.Errorf("Expecting %f but got %f", want, *v)
		}
	}
	if *f7 != float64(7) {
		t.Errorf("Expecting %f but got %f", float64(7), *f7)
	}
	if *f8 != float64(8) {
		t.Errorf("Expecting %f but got %f", float64(8), *f8)
	}
}
