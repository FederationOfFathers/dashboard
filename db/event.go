package db

import (
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

	When           *time.Time   `gorm:"index"`
	Where          string       `gorm:"index"`
	Title          string       `gorm:"type:varchar(191);not null;default:''"`
	Description    string       `gorm:"type:varchar(256);"`
	EventChannel   EventChannel `gorm:"foreignkey:ChannelID"`
	EventChannelID string       `gorm:"type:varchar(191);not null"`
	GUID           string       `gorm:"type:varchar(191);not null;default:'';unique_index"`
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
	ID                  string `gorm:"type:varchar(191);not null;default:'';primary_key" json:"channelID"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time `sql:"index"`
	ChannelCategoryName string     `json:"categoryName"`
	ChannelCategoryID   string     `json:"categoryID"`
	ChannelName         string     `json:"name"`
	db                  *DB        `gorm:"-"`
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
	if err != nil && err != gorm.ErrRecordNotFound {
		return e, err
	}

	for _, event := range e {
		event.db = d
		d.Raw("SELECT * FROM event_members WHERE event_id = ?", event.ID).Scan(&event.Members)
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
	err := d.Where(&EventChannel{ID: chID}).First(&eventChannel).Error
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
	event.ID = uint(id)
	err := d.Raw("SELECT * FROM events WHERE events.id = ? LIMIT 1", id).Scan(event).Error

	event.db = d
	return event, err
}

func (d *DB) EventMembers(event *Event) ([]*EventMember, error) {
	var members []*EventMember

	err := d.Raw("SELECT * FROM event_members WHERE event_id = ?", event.ID).Scan(&members).Error

	return members, err
}

func (d *DB) EventMemberByID(id uint) (*EventMember, error) {
	member := EventMember{}

	err := d.Raw("SELECT * FROM event_members WHERE id = ? LIMIT 1", id).Scan(&member).Error

	return &member, err

}

func (d *DB) DeleteEventMemberByID(u uint) {
	if err := d.Exec("DELETE FROM event_members WHERE id = ?", u).Error; err != nil {
		Logger.Error("unable to delete event members", zap.Uint("id", u), zap.Error(err))
	}
}

// SaveEventChannel creates or saves an EventChannel
func (d *DB) SaveEventChannel(e *EventChannel) error {

	existingCh := &EventChannel{}
	err := d.Where("id = ?", e.ID).Find(&existingCh).Error

	// create
	if err == gorm.ErrRecordNotFound {
		Logger.Info("new guild channel", zap.String("id", e.ID), zap.String("name", e.ChannelName))
		return d.Create(e).Error
	}

	// update
	existingCh.ChannelName = e.ChannelName
	existingCh.ChannelCategoryName = e.ChannelCategoryName
	existingCh.ChannelCategoryID = e.ChannelCategoryID

	return d.Save(&existingCh).Error

}

// DeleteEvent delete event
func (d *DB) DeleteEvent(e Event) {

	//delete the members
	if err := d.Exec("DELETE FROM event_members WHERE event_id = ?", e.ID).Error; err != nil {
		Logger.Error("unable to delete event members", zap.Uint("event_id", e.ID), zap.Error(err))
	}

	// delete the event
	if err := d.Unscoped().Delete(&e).Error; err != nil {
		Logger.Error("unable to delete event", zap.Uint("event_id", e.ID), zap.Error(err))
	}
}

// PurgeOldEventChannels purge event channels that have not been updated in the given amount of time
func (d *DB) PurgeOldEventChannels(t time.Duration) { //TODO fix this
	Logger.Info("Purging old channels", zap.Duration("duration", t))
	now := time.Now()
	now = now.Add(t)
	d.Unscoped().Where("updated_at < ?", now).Delete(&EventChannel{})
}
