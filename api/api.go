package api

import (
	"net/http"
	"os"
	"regexp"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

var ListenOn = ":8866"
var Router = mux.NewRouter()
var logger = zap.NewExample().With(zap.String("module", "api")).Sugar()

var DB *db.DB

var allowedOrigins = regexp.MustCompile(`^https?://((localhost|127\.0\.0\.1)(:[0-9]+)?|([^.]*\.)?fofgaming.com)$`)

func Run() {
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
