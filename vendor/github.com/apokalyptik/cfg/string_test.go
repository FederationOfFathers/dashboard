package cfg

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestString(t *testing.T) {
	os.Setenv("STRING_S1", "1 is 1")
	ioutil.WriteFile(".string.json", []byte(`{"s2": "2 is 2"}`), 0644)
	ioutil.WriteFile(".string.yml", []byte("---\ns3: 3 is 3\ns4: \"4 is 4\""), 0644)
	c := New("string")
	vs := []*string{
		c.String("s1", "1 is 1", ""),
		c.String("s2", "2 is 2", ""),
		c.String("s3", "3 is 3", ""),
		c.String("s4", "4 is 4", ""),
		c.String("s5", "5 is 5", ""),
	}
	for k, v := range vs {
		want := fmt.Sprintf("%d is %d", (k + 1), (k + 1))
		if *v != want {
			t.Errorf("Expecting %s but got %s", want, *v)
		}
	}
}
