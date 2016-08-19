package roster

import (
	"github.com/FederationOfFathers/dashboard/store"
	stow "gopkg.in/djherbis/stow.v2"
)

var db *stow.Store

func Init() {
	db = store.DB.Friends()
}

func member(userID string) *stow.Store {
	return db.NewNestedStore([]byte(userID))
}

func Get(userID string) []string {
	var friends = []string{}
	member(userID).ForEach(func(userID string, status bool) {
		if status {
			friends = append(friends, userID)
		}
	})
	return friends
}

func Set(userID, friendID string, friends bool) error {
	return member(userID).Put(friendID, friends)
}
