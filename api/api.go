package api

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"net/http"
	"os"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"go.uber.org/zap"
)

var ListenOn = ":8866"
var Router = mux.NewRouter()
var Logger *zap.Logger

var DB *db.DB

func Run() {
	jwtSecretBytes = []byte(JWTSecret)
	s := sha1.New()
	m := md5.New()
	s.Write(jwtSecretBytes)
	m.Write(jwtSecretBytes)
	cookie = securecookie.New(s.Sum(nil), m.Sum(nil))
	Logger.Fatal(
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
							handlers.AllowedOrigins([]string{
								"http://ui.fofgaming.com",
								"https://ui.fofgaming.com",
								"http://dev.fofgaming.com",
								"https://dev.fofgaming.com",
								"http://127.0.0.1:3000",
								"http://localhost:3000",
								"http://127.0.0.1",
								"http://localhost",
							}),
						)(Router),
					),
				),
			),
		)))
}

func NotImplemented(w http.ResponseWriter, e *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprint(w, "Not Implemented")
}
