package api

import (
	"net/http"
	"os"
	"regexp"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/uber-go/zap"
)

var ListenOn = ":8866"
var Router = mux.NewRouter()
var logger = zap.New(zap.NewJSONEncoder(zap.RFC3339Formatter("time"))).With(zap.String("module", "api"))

var URLHostName = ""
var URLPrefix = ""
var URLScheme = "https"
var DB *db.DB

var allowedOrigins = regexp.MustCompile(`^https?://((localhost|127\.0\.0\.1)(:[0-9]+)?|([^.]*\.)?fofgaming.com)$`)

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
		zap.Error(http.ListenAndServe(ListenOn,
			handlers.ProxyHeaders(
				handlers.CompressHandler(
					handlers.CombinedLoggingHandler(os.Stdout,
						handlers.CORS(
							handlers.AllowCredentials(),
							handlers.AllowedHeaders([]string{
								"Content-Type",
								"Authorization",
								"X-Requested-With",
							}),
							handlers.AllowedMethods([]string{
								"GET",
								"HEAD",
								"PUT",
								"POST",
								"DELETE",
							}),
							handlers.AllowedOriginValidator(allowedOrigins.MatchString),
						)(Router),
					),
				),
			),
		)))
}
