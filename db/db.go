package db

import (
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type DB struct {
	*gorm.DB
	eventCacheLock sync.RWMutex
	eventCache     []*Event
}

func (d *DB) migrate() {
	d.DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&Member{})
	d.DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&MemberMeta{})
	d.DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&Stream{})
	d.DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&Event{})
	d.DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&EventMember{})
	d.DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&EventChannel{})
	d.DB.Exec("DROP TABLE logins")
	d.DB.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci").AutoMigrate(&Logins{})
}

func New(dialect string, args ...interface{}) *DB {
	d, err := gorm.Open(dialect, args...)
	if err != nil {
		panic(err)
	}
	var rval = &DB{
		DB: d,
	}
	rval.eventCache = []*Event{}
	rval.LogMode(false)
	rval.migrate()
	return rval
}
