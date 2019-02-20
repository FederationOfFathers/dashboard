package streams

import (
	"fmt"
	"sync"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
	"go.uber.org/zap"
)

var Streams = []*db.Stream{}
var lock sync.Mutex
var Logger *zap.Logger
var channel string

var DB *db.DB

func Init(notifySlackChannel string) error {
	channel = notifySlackChannel
	updated()
	return nil
}

func updated() {
	if s, err := DB.Streams(); err != nil {
		Logger.Error("Error updating streams", zap.Error(err))
	} else {
		Streams = s
	}
}

func Mind() {
	go mind()
}

func MindList() {
	go func() {
		uptimer := time.Tick(1 * time.Minute)
		for {
			select {
			case <-uptimer:
				updated()
			}
		}
	}()
}

func mind() {
	mindYoutube()
	mindTwitch()
	mindMixer()
	uptimer := time.Tick(1 * time.Minute)
	twtimer := time.Tick(1 * time.Minute)
	yttimer := time.Tick(1 * time.Minute)
	bptimer := time.Tick(1 * time.Minute)
	for {
		select {
		case <-uptimer:
			updated()
		case <-twtimer:
			mindTwitch()
		case <-yttimer:
			mindYoutube()
		case <-bptimer:
			mindMixer()
		}
	}
}

func Owner(s *db.Stream) (*db.Member, error) {
	return DB.MemberByID(s.MemberID)
}

func Add(kind, identifier, userID string) error {
	member, err := DB.MemberByAny(userID)
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
	case "beam":
		err := DB.Exec(
			"INSERT INTO `streams` (`member_id`,`beam`) VALUES (?,?) ON DUPLICATE KEY UPDATE `beam`=?",
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
	case "beam":
		err := DB.Exec("UPDATE `streams` SET `beam` = '' WHERE `id` = ?", memberID).Error
		updated()
		return err
	case "youtube":
		err := DB.Exec("UPDATE `streams` SET `youtube` = '' WHERE `id` = ?", memberID).Error
		updated()
		return err
	}
	return fmt.Errorf("unknown kind!")
}
