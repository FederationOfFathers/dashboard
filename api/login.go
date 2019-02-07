package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
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

	// v0 using slack
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

	// V1 using member id instead of slack
	Router.Path("/api/v1/login/check/{code}").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		if err := DB.Exec("DELETE FROM logins WHERE `expiry` < NOW()").Error; err != nil {
			log.Error("cleaning logins", zap.Error(err))
		}

		code := mux.Vars(r)["code"]
		login, err := DB.GetLoginForCode(code)

		// handle errors
		if err != nil {
			// no record = already used the code
			if err == gorm.ErrRecordNotFound {
				json.NewEncoder(w).Encode("gone")
				return
			}
			log.Error("login code check", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// record found, redirect to login
		if login.MemberID != 0 {
			var link = fmt.Sprintf("/api/v1/login?w=%d&t=%s&r=0", login.MemberID, GenerateValidAuthTokens(strconv.Itoa(login.MemberID))[0])
			DB.DeleteLoginForCode(code)
			http.Redirect(w, r, link, http.StatusTemporaryRedirect)
			return
		}

		// keep waiting
		json.NewEncoder(w).Encode("wait")
	})

	// LOGOUT
	Router.Path("/api/v0/logout").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		http.SetCookie(
			w,
			&http.Cookie{
				Name:     "secure-cookie",
				Value:    "",
				Path:     "/",
				Domain:   "fofgaming.com",
				Expires:  time.Now().Add(-365 * 24 * time.Hour), // -365 in order to subtract 1 year
				HttpOnly: false,
			},
		)
		json.NewEncoder(w).Encode("logout complete")
	})

}
