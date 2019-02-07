package db

import (
	"time"

	"go.uber.org/zap"
)

type Logins struct {
	ID       int    `gorm:"type:int(11);not null;auto_increment;primary_key"`
	Member   string `gorm:"type:varchar(191);null"`
	MemberID int
	code     string `gorm:"type:varchar(8};not null"`
	expiry   time.Time
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
