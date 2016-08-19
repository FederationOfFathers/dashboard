package events

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/uber-go/zap"
)

var logger = zap.NewJSON().With(zap.String("module", "event"))
var Data = &Events{}
var slackData = bridge.Data.Slack
var SaveFile = "events.json"
var SaveInterval = time.Minute

type Events struct {
	sync.RWMutex
	saved   bool
	started bool
	list    []*Event
}

func (e *Events) FindForSlackUserID(userID string) []*Event {
	var whereIDs = []string{}
	for _, c := range slackData.UserChannels(userID) {
		whereIDs = append(whereIDs, c.ID)
	}
	for _, g := range slackData.UserGroups(userID) {
		whereIDs = append(whereIDs, g.ID)
	}
	var rval = []*Event{}
	e.RLock()
	for _, ev := range e.list {
		for _, id := range whereIDs {
			if id == ev.ID {
				rval = append(rval, ev)
				break
			}
		}
	}
	e.RUnlock()
	return rval
}

func (e *Events) load() {
	e.Lock()
	defer e.Unlock()
	fp, err := os.Open(SaveFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		logger.Fatal("Unable to open savefile for reading", zap.String("filename", SaveFile), zap.Error(err))
	}
	defer fp.Close()
	dec := json.NewDecoder(fp)
	var version int
	if err := dec.Decode(&version); err != nil {
		logger.Fatal("error decoding version number", zap.Error(err))
	}
	// if we change the datafile format here is where we would do conversion.
	var i = 0
	for dec.More() {
		i = i + 1
		ev := new(Event)
		if err := dec.Decode(ev); err != nil {
			logger.Fatal("error decoding savefile", zap.String("filename", SaveFile), zap.Error(err), zap.Int("record", i))
			break
		}
		e.list = append(e.list, ev)
	}
}

func (e *Events) save() {
	tempfile := fmt.Sprintf(".%s.tmp", SaveFile)
	e.Lock()
	defer e.Unlock()
	if e.saved {
		logger.Debug("no need to save events")
		return
	}
	fp, err := os.Create(tempfile)
	if err != nil {
		logger.Fatal("Unable to open savefile for writing", zap.String("filename", tempfile), zap.Error(err))
	}
	defer fp.Close()
	enc := json.NewEncoder(fp)
	if err := enc.Encode(1); err != nil {
		logger.Fatal("error encoding version number", zap.Error(err))
	}
	for _, event := range e.list {
		if err := enc.Encode(*event); err != nil {
			logger.Fatal("error encoding event", zap.String("filename", tempfile), zap.Error(err))
		}
	}
	if err := os.Rename(tempfile, SaveFile); err != nil {
		logger.Fatal("error renaming temporary save file", zap.String("from", tempfile), zap.String("to", SaveFile), zap.Error(err))
	}
	logger.Info("data saved")
	e.saved = true
}

func (e *Events) childUpdate() {
	if e == nil {
		return
	}
	e.Lock()
	e.saved = false
	e.Unlock()
	logger.Debug("notified of save requirement")
}

func (e *Events) AddEvent(ev *Event) {
	e.Lock()
	e.list = append(e.list, ev)
	e.Unlock()
	e.childUpdate()
}

func Start() {
	logger.SetLevel(zap.DebugLevel)
	Data.Lock()
	if Data.started {
		Data.Unlock()
		return
	}
	Data.started = true
	Data.saved = true
	Data.Unlock()
	Data.load()
	go func() {
		t := time.Tick(SaveInterval)
		for {
			select {
			case <-t:
				Data.save()
			}
		}
	}()
}
