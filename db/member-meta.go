package db

import "time"

type MemberMeta struct {
	ID        int    `sql:"bigint(20) NOT NULL AUTO_INCREMENT"`
	MemberID  int    `sql:"bigint(20) NOT NULL" gorm:"unique_index:user_meta_key"`
	MetaKey   string `gorm:"type:varchar(191);not null;unique_index:user_meta_key;index:meta_key"`
	MetaJSON  []byte
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	db        *DB `gorm:"-"`
}

func (m *MemberMeta) Save() error {
	return m.db.Save(m).Error
}
