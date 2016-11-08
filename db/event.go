package db

import (
	"log"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/jinzhu/gorm"
)

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
	if e.db.NewRecord(e) {
		return e.db.Create(&e).Error
	}
	return e.db.Save(&e).Error
}

func (e *Event) BeforeCreate() error {
	e.GUID = uuid.New()
	return nil
}
