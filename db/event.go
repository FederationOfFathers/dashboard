package db

import (
	"log"
	"sync"
	"time"

	"github.com/pborman/uuid"

	"github.com/jinzhu/gorm"
)

var wait = sync.NewCond(&sync.Mutex{})

func (d *DB) EventsUpdated() ([]*Event, error) {
	wait.L.Lock()
	wait.Wait()
	wait.L.Unlock()
	d.eventCacheLock.RLock()
	if d.eventCache != nil {
		defer d.eventCacheLock.RUnlock()
		return d.eventCache, nil
	}
	d.eventCacheLock.RUnlock()
	d.eventCacheLock.Lock()
	defer d.eventCacheLock.Unlock()
	if d.eventCache != nil {
		// Populated during race
		return d.eventCache, nil
	}
	events, err := d.Events()
	if err == nil {
		d.eventCache = events
	}
	return d.eventCache, nil
}

type Event struct {
	gorm.Model
	db *DB `gorm:"-"`

	When    *time.Time `gorm:"index"`
	Where   string     `gorm:"index"`
	Title   string     `gorm:"type:varchar(191);not null;default:''"`
	GUID    string     `gorm:"type:varchar(191);not null;default:'';unique_index"`
	Members []EventMember
}

type EventMember struct {
	gorm.Model

	Type    int
	EventID uint
	Member  Member
}

func (d *DB) NewEvent() *Event {
	return &Event{
		db: d,
	}
}

func (d *DB) Events() ([]*Event, error) {
	var e []*Event
	err := d.Find(&e).Error
	for _, event := range e {
		log.Printf("%#v", event)
		event.db = d
		event.db.Model(event).Related(&event.Members, "EventMembers")
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return e, err
	}
	return e, nil
}

func (e *Event) Save() error {
	var err error

	if e.db.NewRecord(e) {
		err = e.db.Create(&e).Error
	} else {
		err = e.db.Save(&e).Error
	}
	if err == nil {
		wait.L.Lock()
		e.db.eventCacheLock.Lock()
		if events, err := e.db.Events(); err == nil {
			e.db.eventCache = events
		} else {
			log.Printf("Error updating event cache after save: %s", err.Error())
			e.db.eventCache = nil
		}
		e.db.eventCacheLock.Unlock()
		wait.Broadcast()
		wait.L.Unlock()
	}
	return err
}

func (e *Event) BeforeCreate() error {
	e.GUID = uuid.New()
	return nil
}
