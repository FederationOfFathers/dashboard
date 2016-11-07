package db

import "time"

type Stream struct {
	ID             int       `sql:"bigint(20) NOT NULL AUTO_INCREMENT"`
	MemberID       int       `gorm:"type:bigint;not null;default:0;unique_index"`
	Twitch         string    `gorm:"type:varchar(191);not null;default:'';index"`
	TwitchStreamID string    `gorm:"type:varchar(191);not null;default:''"`
	TwitchStart    int64     `gorm:"type:bigint;not null;default:0"`
	TwitchStop     int64     `gorm:"type:bigint;not null;default:0"`
	Youtube        string    `gorm:"type:varchar(191);not null;default:'';index"`
	YoutubeStart   int64     `gorm:"type:bigint;not null;default:0"`
	YoutubeStop    int64     `gorm:"type:bigint;not null;default:0"`
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
