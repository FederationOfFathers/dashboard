package db

import "time"

type Member struct {
	ID      int    `sql:"bigint(20) NOT NULL AUTO_INCREMENT"`
	Slack   string `gorm:"type:varchar(191);not null;default:'';unique_index"`
	Xbl     string `gorm:"type:varchar(191);not null;default:'';index"`
	Psn     string `gorm:"type:varchar(191);not null;default:''"`
	Destiny string `gorm:"type:varchar(191);not null;default:''"`
	Seen    int    `gorm:"type:bigint;not null;index;default:0"`
	Name    string `gorm:"type:varchar(191);not null;default:''"`
	TZ      string `gorm:"type:varchar(191);not null;default:''"`
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
