package streams

import (
	"fmt"
	"sync"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/uber-go/zap"
)

var Streams = []*db.Stream{}
var lock sync.Mutex
var logger = zap.New(zap.NewJSONEncoder()).With(zap.String("module", "streams"))
var channel string

var DB *db.DB

func Init(notifySlackChannel string) error {
	channel = notifySlackChannel
	updated()
	return nil
}

func updated() {
	if s, err := DB.Streams(); err != nil {
		logger.Error("Error updating streams", zap.Error(err))
	} else {
		Streams = s
	}
}

func Mind() {
	go mind()
}

func mind() {
	mindYoutube()
	mindTwitch()
	uptimer := time.Tick(5 * time.Minute)
	twtimer := time.Tick(5 * time.Minute)
	yttimer := time.Tick(5 * time.Minute)
	for {
		select {
		case <-uptimer:
			updated()
		case <-twtimer:
			mindTwitch()
		case <-yttimer:
			mindYoutube()
		}
	}
}

func Owner(s *db.Stream) (*db.Member, error) {
	return DB.MemberByID(s.MemberID)
}

func Add(kind, identifier, userID string) error {
	member, err := DB.MemberBySlackID(userID)
	if err != nil {
		return err
	}
	switch kind {
	case "twitch":
		err := DB.Exec(
			"INSERT INTO `streams` (`member_id`,`twitch`) VALUES (?,?) ON DUPLICATE KEY UPDATE `twitch`=?",
			member.ID,
			identifier,
			identifier,
		).Error
		updated()
		return err
	case "youtube":
		err := DB.Exec(
			"INSERT INTO `streams` (`member_id`,`youtube`) VALUES (?,?) ON DUPLICATE KEY UPDATE `youtube`=?",
			member.ID,
			identifier,
			identifier,
		).Error
		updated()
		return err
	}
	return fmt.Errorf("unknown kind!")
}

func Remove(memberID int, kind string) error {
	switch kind {
	case "twitch":
		err := DB.Exec("UPDATE `streams` SET `twitch` = '' WHERE `id` = ?", memberID).Error
		updated()
		return err
	case "youtube":
		err := DB.Exec("UPDATE `streams` SET `youtube` = '' WHERE `id` = ?", memberID).Error
		updated()
		return err
	}
	return fmt.Errorf("unknown kind!")
}
