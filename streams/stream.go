package streams

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/FederationOfFathers/dashboard/store"
	"github.com/levi/twch"
	"github.com/uber-go/zap"
	stow "gopkg.in/djherbis/stow.v2"
)

var Streams = map[string]*Stream{}

var lock sync.Mutex

var logger = zap.NewJSON().With(zap.String("module", "streams"))
var twlog = zap.NewJSON().With(zap.String("module", "streams"), zap.String("service", "twitch"))
var ytlog = zap.NewJSON().With(zap.String("module", "streams"), zap.String("service", "twitch"))

var TwitchOAuthKey string
var twitchClient *twch.Client

var YoutubeAPIKey string

var db *stow.Store
var channel string

type Stream struct {
	db        *stow.Store
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

func (s *Stream) updateYoutube() {
}

func (s *Stream) updateTwitch() {
	now := time.Now()
	stream, _, err := twitchClient.Streams.GetStream(s.ServiceID)
	if err != nil {
		twlog.Error("error fetching stream", zap.String("key", s.ServiceID), zap.Error(err))
		return
	}

	bs, _ := json.MarshalIndent(stream, "", "  ")
	log.Println(string(bs))

	if stream.ID == nil {
		s.Stop = &now
		// TODO detect stopped stream
		return
	}
	if s.Twitch == nil {
		s.Twitch = &twch.Stream{}
	}

	if s.Twitch.ID != nil {
		if *s.Twitch.ID == *stream.ID {
			twlog.Debug("still streaming", zap.String("key", s.ServiceID))
			return
		}
		s.Twitch = stream
		then := now.Add(0 - time.Second)
		s.Stop = &then
		s.Start = &now
		twlog.Debug("stopped and then started again", zap.String("key", s.ServiceID))
		b, _ := json.MarshalIndent(s, "", "  ")
		log.Println(string(b))
		s.db.Put(s.Key(), s)
		return
	}
	s.Start = &now
	s.Twitch = stream
	twlog.Debug("started", zap.String("key", s.ServiceID))
	b, _ := json.MarshalIndent(s, "", "  ")
	log.Println(string(b))
	s.db.Put(s.Key(), s)
}

func Init(notifySlackChannel string) error {
	logger.SetLevel(zap.DebugLevel)
	twlog.SetLevel(zap.DebugLevel)
	ytlog.SetLevel(zap.DebugLevel)
	db = store.DB.Streams()
	channel = notifySlackChannel
	tclient, err := twch.NewClient(TwitchOAuthKey, nil)
	if err != nil {
		return err
	}
	twitchClient = tclient
	db.ForEach(func(key string, value *Stream) {
		value.db = db
		Streams[key] = value
	})
	go mind()
	return nil
}

func mindTwitch() {
	twlog.Debug("begin minding")
	for key, stream := range Streams {
		if stream.Kind != "twitch" {
			continue
		}
		twlog.Debug("minding", zap.String("key", key))
		stream.update()
	}
	twlog.Debug("end minding")
}

func mindYoutube() {
	ytlog.Debug("begin minding")
	for key, stream := range Streams {
		if stream.Kind != "youtube" {
			continue
		}
		ytlog.Debug("minding", zap.String("key", key))
		stream.update()
	}
	ytlog.Debug("end minding")
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
	if err := db.Put(key, s); err != nil {
		logger.Error("error adding", zap.String("key", key), zap.Error(err))
		return err
	}
	Streams[key] = s
	logger.Info("added", zap.String("key", key))
	return nil
}

func Remove(kind, identifier string) error {
	key := fmt.Sprintf("%s:%s", kind, identifier)
	if err := db.Delete(key); err != nil {
		logger.Error("error deleting", zap.String("key", key), zap.Error(err))
		return err
	}
	delete(Streams, key)
	logger.Info("deleted", zap.String("key", key))
	return nil
}
