package db

import "time"

type Logins struct {
	ID       int    `gorm:"type:int(11);not null;auto_increment;primary_key"`
	Member   string `gorm:"type:varchar(191);null"`
	MemberID int
	code     string `gorm:"type:varchar(8};not null"`
	expiry   time.Time
}

// GetLoginForCode returns a Logins{} where the code matches and an error if any
func (d *DB) GetLoginForCode(code string) (Logins, error) {

	login := Logins{}
	err := d.Where("code =?", code).First(&login).Error
	return login, err
}

// DeleteLoginForCode deletes login rows with the matching code
func (d *DB) DeleteLoginForCode(code string) {
	d.Where("code = ?", code).Delete(Logins{})
}
