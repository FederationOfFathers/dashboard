package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/labstack/gommon/log"
	uuid "github.com/nu7hatch/gouuid"
	hashids "github.com/speps/go-hashids"
	"go.uber.org/zap"
)

var hashID *hashids.HashID
var hashIDPlace int
var hashIDLock sync.Mutex

func loginToken() string {
	hashIDLock.Lock()
	var n = hashIDPlace
	hashIDPlace++
	hashIDLock.Unlock()
	s, _ := hashID.Encode([]int{n})
	s = strings.ToLower(s)
	return s
}

func init() {
	u4, _ := uuid.NewV4()
	var hd = hashids.NewData()
	hd.Salt = u4.String()
	hd.MinLength = 4
	hashids.NewWithData(hd)
	hashID = hashids.NewWithData(hd)

	Router.Path("/api/v0/login/get").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := DB.Exec("DELETE FROM logins WHERE `expiry` < NOW()").Error; err != nil {
			log.Error("cleaning logins", zap.Error(err))
		}
		s := loginToken()
		err := DB.Exec(
			"INSERT INTO logins (`code`,`expiry`) VALUES(?,DATE_ADD(NOW(), INTERVAL 15 MINUTE))",
			s,
		).Error
		if err != nil {
			Logger.Error("inserting", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(s)
	})
	Router.Path("/api/v0/login/check/{code}").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		if err := DB.Exec("DELETE FROM logins WHERE `expiry` < NOW()").Error; err != nil {
			log.Error("cleaning logins", zap.Error(err))
		}
		var who string
		row := DB.Raw("SELECT member FROM logins WHERE code = ?", mux.Vars(r)["code"]).Row()
		if err := row.Scan(&who); err != nil {
			if err == sql.ErrNoRows {
				json.NewEncoder(w).Encode("gone")
				return
			}
			log.Error("scanning", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if who != "" {
			var link = fmt.Sprintf("/api/v0/login?w=%s&t=%s&r=0", who, GenerateValidAuthTokens(who)[0])
			DB.Exec("DELETE FROM logins WHERE code = ? LIMIT 1", mux.Vars(r)["code"])
			http.Redirect(w, r, link, http.StatusTemporaryRedirect)
			return
		}
		json.NewEncoder(w).Encode("wait")
	})

}