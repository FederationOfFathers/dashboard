package events

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/url"
	"time"
)

var OldEventLinkHMAC string

func OldEventToolAuthorization(username string) string {
	mac := hmac.New(sha256.New, []byte(OldEventLinkHMAC))
	t := time.Now()
	fmt.Fprintln(mac, username, t.Unix())
	return fmt.Sprintf("%s:%d:%s", username, t.Unix(), fmt.Sprintf("%x", mac.Sum(nil)))
}

func OldEventToolLink(username string) string {
	mac := hmac.New(sha256.New, []byte(OldEventLinkHMAC))
	t := time.Now()
	fmt.Fprintln(mac, username, t.Unix())
	return fmt.Sprintf(
		"//team.fofgaming.com/rest/login?username=%s&t=%d&signature=%s",
		url.QueryEscape(username),
		t.Unix(),
		fmt.Sprintf("%x", mac.Sum(nil)),
	)
}
