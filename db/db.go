package db

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type DB struct {
	*gorm.DB
}

func (d *DB) migrate() {
	d.DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&Member{})
	d.DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&Stream{})
}

func New(dialect string, args ...interface{}) *DB {
	d, err := gorm.Open(dialect, args...)
	if err != nil {
		panic(err)
	}
	var rval = &DB{
		DB: d,
	}
	rval.migrate()
	return rval
}
