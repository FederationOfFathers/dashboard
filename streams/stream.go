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

var DB *db.DB

func updated() {
	if s, err := DB.Streams(); err != nil {
		Logger.Error("Error updating streams", zap.Error(err))
	} else {
		Logger.Debug("streams updated", zap.Int("streams", len(s)))
		Streams = s
	}
}

func Mind() {
	go mind()
}

// starts the minder that checks and updates stream messages every minute
func mind() {
	updateAndMind()
	minuteTicker := time.Tick(1 * time.Minute)
	for {
		select {
		case <-minuteTicker:
			updateAndMind()
		}
	}
}

// keeping all the updates together in order to avoid reading while writing
func updateAndMind() {
	updated()
	mindYoutube()
	mindTwitch()
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
	return fmt.Errorf("unknown kind '%s'", kind)
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
