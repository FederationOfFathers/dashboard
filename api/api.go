package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/gorilla/mux"
	"github.com/uber-go/zap"
)

var ListenOn = ":8866"
var Router = mux.NewRouter()
var logger = zap.New(zap.NewJSONEncoder()).With(zap.String("module", "api"))

var URLHostName = ""
var URLPrefix = ""
var URLScheme = "https"
var DB *db.DB

func myURL() string {
	return fmt.Sprintf("%s://%s:%s%s", URLScheme, URLHostName, ListenOn, URLPrefix)
}

func Run() {
	if URLHostName == "" {
		URLHostName, _ = os.Hostname()
	}
	logger.Fatal(
		"error starting API http server",
		zap.String("listenOn", ListenOn),
		zap.Error(http.ListenAndServe(ListenOn, Router)))
}
