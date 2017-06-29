package api

import (
	"net/http"
	"os"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/gorilla/mux"
	"github.com/uber-go/zap"
)

var ListenOn = ":8866"
var Router = mux.NewRouter()
var logger = zap.New(zap.NewJSONEncoder(zap.RFC3339Formatter("time"))).With(zap.String("module", "api"))

var URLHostName = ""
var URLScheme = "https"
var DB *db.DB
var UseHttps = false

func InitURLScheme() {
	if UseHttps {
		URLScheme = "https"
	} else {
		URLScheme = "http"
	}
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
