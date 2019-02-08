package db

import (
	"time"

	"go.uber.org/zap"
)

type Logins struct {
	Code     string    `gorm:"type:varchar(191);not null;default:'';primary_key"`
	Member   string    `gorm:"type:varchar(191);null"`
	MemberID int       `gorm:"type:int(11);null"`
	Expiry   time.Time `gorm:"not null;default:'1970-01-01 00:00:01';index:expiry"`
}

// GetLoginForCode returns a Logins{} where the code matches and an error if any
func (d *DB) GetLoginForCode(code string) (Logins, error) {

	var login Logins
	err := d.Raw("SELECT * FROM logins WHERE code = ? LIMIT 1", code).Scan(&login).Error
	return login, err
}

// DeleteLoginForCode deletes login rows with the matching code
func (d *DB) DeleteLoginForCode(code string) {
	Logger.Debug("Deleteing", zap.String("code", code))
	d.Exec("DELETE FROM logins WHERE code = ?", code)
}
