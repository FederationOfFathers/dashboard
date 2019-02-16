package db

import (
	"regexp"
	"strconv"
	"time"
)

var cTypeDigit = regexp.MustCompile("^[0-9]+$")

type Member struct {
	ID      int    `sql:"bigint(20) NOT NULL AUTO_INCREMENT" json:"id"`
	Slack   string `gorm:"type:varchar(191);null;unique_index" json:"slack_id"`
	Discord string `gorm:"type:varchar(191);null;unique_index" json:"discord"`
	Xbl     string `gorm:"type:varchar(191);not null;default:'';index" json:"-"`
	Psn     string `gorm:"type:varchar(191);not null;default:''" json:"-"`
	Destiny string `gorm:"type:varchar(191);not null;default:''" json:"-"`
	Seen    int    `gorm:"type:bigint;not null;index;default:0" json:"-"`
	Name    string `gorm:"type:varchar(191);not null;default:''" json:"name"`
	TZ      string `gorm:"type:varchar(191);not null;default:''" json:"-"`
	db      *DB    `gorm:"-"`
}

func (m *Member) Save() error {
	rval := m.db.Save(m)
	return rval.Error
}

func (d *DB) MemberByAny(some string) (*Member, error) {
	if cTypeDigit.MatchString(some) {
		i, err := strconv.Atoi(some)
		if err != nil {
			return nil, err
		}
		return d.MemberByID(i)
	}
	if member, _ := d.MemberBySlackID(some); member != nil {
		return member, nil
	}
	return d.MemberByName(some)
}

func (d *DB) MemberByID(id int) (*Member, error) {
	m := new(Member)
	err := d.First(&m, id).Error
	m.db = d
	return m, err
}

func (d *DB) MemberByDiscordID(id string) (*Member, error) {
	m := new(Member)
	err := d.DB.Where("discord = ?", id).First(&m).Error
	m.db = d
	return m, err
}

func (d *DB) MemberByName(name string) (*Member, error) {
	m := new(Member)
	err := d.DB.Where("name = ?", name).First(&m).Error
	m.db = d
	return m, err
}

func (d *DB) MemberBySlackID(id string) (*Member, error) {
	m := new(Member)
	err := d.DB.Where("slack = ?", id).First(&m).Error
	m.db = d
	return m, err
}

func (d *DB) MembersActive(since time.Time) ([]*Member, error) {
	m := []*Member{}
	err := d.DB.Where("seen >= ?", since.Unix()).Find(&m).Error
	for _, i := range m {
		i.db = d
	}
	return m, err
}

func (d *DB) Members() ([]*Member, error) {
	m := []*Member{}
	err := d.Find(&m).Error
	for _, i := range m {
		i.db = d
	}
	return m, err
}
