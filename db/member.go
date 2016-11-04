package db

import "time"

type Member struct {
	ID      int    `gorm:"primary_key"`
	Name    string ``
	Slack   string `gorm:"index"`
	Xbl     string `gorm:"index"`
	Psn     string ``
	Destiny string ``
	Seen    int    `gorm:"index"`
	db      *DB    `gorm:"-"`
}

func (m *Member) Save() error {
	return m.db.Save(m).Error
}

func (d *DB) MemberByID(id int) (*Member, error) {
	m := new(Member)
	err := d.First(&m, id).Error
	if err != nil {
		m.db = d
	}
	return m, err
}

func (d *DB) MemberByName(name string) (*Member, error) {
	m := new(Member)
	err := d.DB.Where("name = ?", name).First(&m).Error
	if err != nil {
		m.db = d
	}
	return m, err
}

func (d *DB) MemberBySlackID(id string) (*Member, error) {
	m := new(Member)
	err := d.DB.Where("slack = ?", id).First(&m).Error
	if err != nil {
		m.db = d
	}
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
