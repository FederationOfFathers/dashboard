package api

import (
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

var cdnPutKey = os.Getenv("CDN_PUT_KEY")

type cdnData struct {
	Headers http.Header
	Body    []byte
}

func cdnFilePathFor(basePath string, key string) (path string, file string) {
	hasher := md5.New()
	hasher.Write([]byte(key))
	k := hasher.Sum(nil)
	path = fmt.Sprintf("%s/key/%0x/%0x/%0x", basePath, k[:1], k[1:2], k[2:3])
	file = fmt.Sprintf("%s/%0x", path, k)
	return
}

func init() {
	Router.Path("/api/v0/cdn/{key}").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var basePath = "."
		if bot.CdnPath != "" {
			basePath = bot.CdnPath
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, file := cdnFilePathFor(basePath, mux.Vars(r)["key"])
		fp, err := os.Open(file)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer fp.Close()
		var rval *cdnData
		if err := gob.NewDecoder(fp).Decode(&rval); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", rval.Headers.Get("Content-Type"))
		w.Header().Set("Content-Length", rval.Headers.Get("Content-Length"))
		w.Write(rval.Body)
	})
	Router.Path("/api/v0/cdn/{key}").Methods("PUT").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cdnPutKey == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.Header.Get("Access-Key") != cdnPutKey {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		var basePath = "."
		if bot.CdnPath != "" {
			basePath = bot.CdnPath
		} else {
			logger.Info("bot.CdnPath not configures")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Error("ioutil.ReadAll", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		path, file := cdnFilePathFor(basePath, mux.Vars(r)["key"])
		res, err := http.Get(string(body))
		if err != nil {
			Logger.Error("cdn prime fetch", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()
		var fileData = &cdnData{
			Headers: res.Header,
		}
		fileData.Body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			Logger.Error("cdn prime fetch readall", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			if !os.IsExist(err) {
				logger.Error("os.MkdirAll", zap.String("path", path), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		fp, err := os.Create(file)
		if err != nil {
			Logger.Error("cdn prime fetch create", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer fp.Close()
		if err := gob.NewEncoder(fp).Encode(fileData); err != nil {
			Logger.Error("cdn prime fetch encode and write", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}
