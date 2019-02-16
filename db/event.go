package db

import (
	"fmt"
	"sync"
	"time"

	"github.com/pborman/uuid"
	"go.uber.org/zap"

	"github.com/jinzhu/gorm"
)

var Logger *zap.Logger
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

	When           *time.Time `gorm:"index"`
	Where          string     `gorm:"index"`
	Title          string     `gorm:"type:varchar(191);not null;default:''"`
	Description    string     `gorm:"type:varchar(256);"`
	EventChannel   EventChannel
	EventChannelID int
	GUID           string `gorm:"type:varchar(191);not null;default:'';unique_index"`
	Need           int
	Members        []EventMember
}

type EventMember struct {
	gorm.Model

	Type     int
	EventID  uint
	MemberID int
}

type EventChannel struct {
	gorm.Model `json:"-"`

	ChannelID           string `gorm:"type:varchar(191);not null;unique_index" json:"channelID"`
	ChannelCategoryName string `json:"categoryName"`
	ChannelName         string `json:"name"`
	db                  *DB    `gorm:"-"`
}

const (
	// EventMemberTypeHost is the person who created the event
	EventMemberTypeHost int = iota
	// EventMemberTypeMember is a member who is joining the event
	EventMemberTypeMember
	// EventMemberTypeAlt is someone who is joining as a backup or tentatively avaialble
	EventMemberTypeAlt
)

func (d *DB) NewEvent() *Event {
	return &Event{
		db: d,
	}
}

func (d *DB) Events() ([]*Event, error) {
	var e []*Event
	err := d.Find(&e).Error
	for _, event := range e {
		Logger.Debug(fmt.Sprintf("%#v", event))
		event.db = d
		event.db.Model(event).Related(&event.Members, "EventMembers").Related(&event.EventChannel, "EventChannelID")
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
			Logger.Error("Error updating event cache after save", zap.Error(err))
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

// EventChannelByID returns an EventChannel found by the DB ID
func (d *DB) EventChannelByID(id int) (*EventChannel, error) {
	var eventChannel EventChannel
	err := d.Where(id).First(&eventChannel).Error
	return &eventChannel, err
}

// EventChanneByChannelID returns an event channel the by discord channel id (snowflake)
func (d *DB) EventChannelByChannelID(chID string) (*EventChannel, error) {
	var eventChannel EventChannel
	err := d.Where(&EventChannel{ChannelID: chID}).First(&eventChannel).Error
	return &eventChannel, err
}

// EventChannels gets all event channels in the DB, or an error
func (d *DB) EventChannels() ([]EventChannel, error) {
	evChannels := []EventChannel{}
	err := d.Find(&evChannels).Error

	return evChannels, err
}

// EventByID gets an event by the id field
func (d *DB) EventByID(id int) (*Event, error) {
	event := &Event{}
	err := d.Where(id).Find(&event).Error
	event.db = d
	return event, err
}

// SaveEventChannel creates or saves an EventChannel
func (d *DB) SaveEventChannel(e *EventChannel) error {

	existingCh := &EventChannel{}
	d.Where("channel_id = ?", e.ChannelID).Find(&existingCh)

	// create
	if existingCh.ID == 0 {
		Logger.Info("new guild channel", zap.String("id", e.ChannelID), zap.String("name", e.ChannelName))
		return d.Create(e).Error
	}

	// update
	existingCh.ChannelName = e.ChannelName
	existingCh.ChannelCategoryName = e.ChannelCategoryName

	return d.Save(&existingCh).Error

}

// PurgeOldEventChannels purge event channels that have not been updated in the given amount of time
func (d *DB) PurgeOldEventChannels(t time.Duration) {
	Logger.Info("Purging old channels", zap.Duration("duration", t))
	now := time.Now()
	now = now.Add(t)
	d.Unscoped().Where("updated_at < ?", now).Delete(&EventChannel{})
}
