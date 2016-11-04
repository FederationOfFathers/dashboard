package db

import "time"

type Stream struct {
	ID             int       `gorm:"primary_key"`
	MemberID       int       `gorm:"index"`
	Twitch         string    `gorm:"type:varchar(191);index"`
	TwitchStreamID string    `gorm:"type:varchar(191)"`
	TwitchStart    int64     ``
	TwitchStop     int64     ``
	Youtube        string    `gorm:"type:varchar(191);index"`
	YoutubeStart   int64     ``
	YoutubeStop    int64     ``
	CreatedAt      time.Time ``
	UpdatedAt      time.Time ``
	db             *DB       `gorm:"-"`
}

func (s *Stream) Save() error {
	return s.db.Save(s).Error
}

func (d *DB) StreamByID(id int) (*Stream, error) {
	s := new(Stream)
	err := d.First(&s, id).Error
	if err != nil {
		s.db = d
	}
	return s, err
}

func (d *DB) StreamByMemberID(memberID int) (*Stream, error) {
	s := new(Stream)
	err := d.Where("member_id = ?", memberID).First(&s).Error
	if err != nil {
		s.db = d
	}
	return s, err
}

func (d *DB) Streams() ([]*Stream, error) {
	s := []*Stream{}
	err := d.Find(&s, "`twitch` != '' OR `youtube` != ''").Error
	for _, i := range s {
		i.db = d
	}
	return s, err
}
