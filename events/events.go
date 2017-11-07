package events

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/FederationOfFathers/dashboard/bridge"
	"go.uber.org/zap"
)

var Logger *zap.Logger
var Data = &Events{}
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
	for _, c := range bridge.Data.Slack.UserChannels(userID) {
		whereIDs = append(whereIDs, c.ID)
	}
	for _, g := range bridge.Data.Slack.UserGroups(userID) {
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
		Logger.Fatal("Unable to open savefile for reading", zap.String("filename", SaveFile), zap.Error(err))
	}
	defer fp.Close()
	dec := json.NewDecoder(fp)
	var version int
	if err := dec.Decode(&version); err != nil {
		Logger.Fatal("error decoding version number", zap.Error(err))
	}
	// if we change the datafile format here is where we would do conversion.
	var i = 0
	for dec.More() {
		i = i + 1
		ev := new(Event)
		if err := dec.Decode(ev); err != nil {
			Logger.Fatal("error decoding savefile", zap.String("filename", SaveFile), zap.Error(err), zap.Int("record", i))
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
		Logger.Debug("no need to save events")
		return
	}
	fp, err := os.Create(tempfile)
	if err != nil {
		Logger.Fatal("Unable to open savefile for writing", zap.String("filename", tempfile), zap.Error(err))
	}
	defer fp.Close()
	enc := json.NewEncoder(fp)
	if err := enc.Encode(1); err != nil {
		Logger.Fatal("error encoding version number", zap.Error(err))
	}
	for _, event := range e.list {
		if err := enc.Encode(*event); err != nil {
			Logger.Fatal("error encoding event", zap.String("filename", tempfile), zap.Error(err))
		}
	}
	if err := os.Rename(tempfile, SaveFile); err != nil {
		Logger.Fatal("error renaming temporary save file", zap.String("from", tempfile), zap.String("to", SaveFile), zap.Error(err))
	}
	Logger.Info("data saved")
	e.saved = true
}

func (e *Events) childUpdate() {
	if e == nil {
		return
	}
	e.Lock()
	e.saved = false
	e.Unlock()
	Logger.Debug("notified of save requirement")
}

func (e *Events) AddEvent(ev *Event) {
	e.Lock()
	e.list = append(e.list, ev)
	e.Unlock()
	e.childUpdate()
}

func Start() {
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
