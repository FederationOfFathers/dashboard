package streams

import (
	"fmt"
	"sync"
	"time"

	"github.com/FederationOfFathers/dashboard/store"
	"github.com/levi/twch"
	"github.com/uber-go/zap"
)

var Streams = map[string]*Stream{}

var lock sync.Mutex

var logger = zap.NewJSON().With(zap.String("module", "streams"))

var channel string

type Stream struct {
	Kind      string
	UserID    string
	ServiceID string
	Start     *time.Time
	Stop      *time.Time

	Twitch *twch.Stream
}

func (s Stream) Key() string {
	return fmt.Sprintf("%s:%s", s.Kind, s.ServiceID)
}

func (s *Stream) update() {
	switch s.Kind {
	case "twitch":
		s.updateTwitch()
	case "youtube":
		s.updateYoutube()
	}
}

func Init(notifySlackChannel string) error {
	logger.SetLevel(zap.DebugLevel)
	twlog.SetLevel(zap.DebugLevel)
	ytlog.SetLevel(zap.DebugLevel)
	channel = notifySlackChannel
	tclient, err := twch.NewClient(TwitchOAuthKey, nil)
	if err != nil {
		return err
	}
	twitchClient = tclient
	store.DB.Streams().ForEach(func(key string, value *Stream) {
		Streams[key] = value
	})
	go mind()
	return nil
}

func mind() {
	mindYoutube()
	mindTwitch()
	twtimer := time.Tick(5 * time.Minute)
	yttimer := time.Tick(5 * time.Minute)
	for {
		select {
		case <-twtimer:
			mindTwitch()
		case <-yttimer:
			mindYoutube()
		}
	}
}

func Add(kind, identifier, userID string) error {
	lock.Lock()
	defer lock.Unlock()
	var key = fmt.Sprintf("%s:%s", kind, identifier)
	if _, ok := Streams[key]; ok {
		logger.Info("adding idempotently", zap.String("key", key))
		return nil
	}
	s := &Stream{
		Kind:      kind,
		UserID:    userID,
		ServiceID: identifier,
	}
	if err := store.DB.Streams().Put(key, s); err != nil {
		logger.Error("error adding", zap.String("key", key), zap.Error(err))
		return err
	}
	Streams[key] = s
	logger.Info("added", zap.String("key", key))
	return nil
}

func Remove(kind, identifier string) error {
	key := fmt.Sprintf("%s:%s", kind, identifier)
	if err := store.DB.Streams().Delete(key); err != nil {
		logger.Error("error deleting", zap.String("key", key), zap.Error(err))
		return err
	}
	delete(Streams, key)
	logger.Info("deleted", zap.String("key", key))
	return nil
}
