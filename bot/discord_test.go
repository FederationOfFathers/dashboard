package bot

import "testing"

func TestUserIDFromMention(t *testing.T) {
	// check basic user mention
	s := userIDFromMention("<@12345678901234567>")
	if s != "12345678901234567" {
		t.Errorf("basic parsing expected 12345678901234567 but got %s", s)
	}

	// check unavialable user mention
	s = userIDFromMention("<@!987654321098765>")
	if s != "987654321098765" {
		t.Errorf("expected 987654321098765 but got %s", s)
	}
}
