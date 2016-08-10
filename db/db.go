package store

import (
	"compress/gzip"
	"fmt"
	"os"
	"sync"
	"time"

	stow "gopkg.in/djherbis/stow.v2"

	"github.com/boltdb/bolt"
	"github.com/robfig/cron"
)

var DBPath = "database.bolt.db"
var DB = &Store{}

type Store struct {
	*bolt.DB
}

func (s *Store) Slack() *stow.Store {
	return stow.NewJSONStore(s.DB, []byte("slack"))
}

func (s *Store) Streams() *stow.Store {
	return stow.NewJSONStore(s.DB, []byte("streams"))
}

func (s *Store) Friends() *stow.Store {
	return stow.NewJSONStore(s.DB, []byte("friends"))
}

func (s *Store) Groups() *stow.Store {
	return stow.NewJSONStore(s.DB, []byte("groups"))
}

func (s *Store) Events() *stow.Store {
	return stow.NewJSONStore(s.DB, []byte("events"))
}

func (s *Store) Close() error {
	return s.DB.Close()
}

var dbBackupLock sync.Mutex

func doGZBackup(fn string) {
	fh, err := os.Create(fn)
	if err != nil {
		panic(err)
	}
	defer fh.Close()
	gh := gzip.NewWriter(fh)
	defer gh.Close()
	err = DB.DB.View(func(tx *bolt.Tx) error {
		return tx.Copy(gh)
	})
	if err != nil {
		panic(err)
	}
}

func dailyBackup() {
	dbBackupLock.Lock()
	defer dbBackupLock.Unlock()
	for i := range []int{13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1} {
		oldFileName := fmt.Sprintf(".%s.daily.%d.gz", DBPath, i)
		if _, err := os.Stat(oldFileName); os.IsNotExist(err) {
			continue
		}
		os.Rename(oldFileName, fmt.Sprintf(".%s.daily.%d.gz", DBPath, i+1))
	}
	doGZBackup(fmt.Sprintf(".%.daily.%d.gz", DBPath, 1))
}

func hourlyBackupDB() {
	dbBackupLock.Lock()
	defer dbBackupLock.Unlock()
	for i := range []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1} {
		oldFileName := fmt.Sprintf(".%s.hourly.%d.gz", DBPath, i)
		if _, err := os.Stat(oldFileName); os.IsNotExist(err) {
			continue
		}
		os.Rename(oldFileName, fmt.Sprintf(".%s.hourly.%d.gz", DBPath, i+1))
	}
	doGZBackup(fmt.Sprintf(".%s.hourly.%d.gz", DBPath, 1))
}

func Mind() {
	var err error
	DB.DB, err = bolt.Open(DBPath, 0600, &bolt.Options{Timeout: 30 * time.Second})
	if err != nil {
		panic(err)
	}
	c := cron.New()
	c.AddFunc("@hourly", hourlyBackupDB)
	c.AddFunc("@daily", dailyBackup)
	c.Start()
}
