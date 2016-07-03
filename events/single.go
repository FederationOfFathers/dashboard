package events

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/dineshappavoo/basex"
	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
)

type Event struct {
	sync.RWMutex
	list        *Events
	ID          string        `json:"id"`
	NumID       int           `json:"numeric_id"`
	At          time.Time     `json:"at"`
	WhereID     string        `json:"where_slack_id"`
	Where       string        `json:"where"`
	Expiration  time.Duration `json:"expiration"`
	Owner       EventMember   `json:"owner"`
	Members     []EventMember `json:"members"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Created     time.Time     `json:"created"`
	Edited      time.Time     `json:"edited"`
	Audit       []string      `json:"audit_log"`
}

func (e *Event) Join(u slack.User) {
	e.Lock()
	for _, m := range e.Members {
		if u.ID == m.SlackID {
			e.Unlock()
			return
		}
	}
	e.Members = append(e.Members, NewMember(u))
	e.Unlock()
}

func (e *Event) MarshallJSON() ([]byte, error) {
	e.RLock()
	defer e.RUnlock()
	return json.Marshal(e)
}

func (e *Event) Log(str string, args ...interface{}) {
	e.Lock()
	e.Audit = append(e.Audit, fmt.Sprintf(str, args...))
	e.Edited = time.Now()
	e.list.childUpdate()
	e.Unlock()
}

func (e *Event) newID() {
	if idBig, err := rand.Int(rand.Reader, big.NewInt(9223372036854775807)); err != nil {
		logger.Fatal("error generating new id", zap.Error(err))
	} else {
		if idenc, err := basex.Encode(fmt.Sprintf("%d", idBig)); err != nil {
			logger.Fatal("error encoding neq event id", zap.Error(err))
		} else {
			e.Lock()
			e.NumID = int(idBig.Int64())
			e.ID = idenc
			e.Unlock()
			e.Log("ID Generated: %s", idenc)
		}
	}
}

func NewEvent() *Event {
	rval := &Event{
		Expiration: 6 * time.Hour,
		Created:    time.Now(),
		Edited:     time.Now(),
		Members:    []EventMember{},
		Audit:      []string{},
	}
	rval.newID()
	rval.Log("Created")
	return rval
}
