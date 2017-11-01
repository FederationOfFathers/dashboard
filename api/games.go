package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func init() {
	Router.Path("/api/v0/games/played/top/{days}/{number}.json").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			n, _ := strconv.Atoi(mux.Vars(r)["number"])
			d, _ := strconv.Atoi(mux.Vars(r)["days"])
			rows, err := DB.Raw(
				"SELECT g.Name as game, COUNT(mg.member) as players "+
					"FROM membergames mg JOIN games g ON( mg.game = g.id ) "+
					"WHERE mg.played > DATE_SUB(NOW(), INTERVAL ? DAY) "+
					"GROUP BY g.Name ORDER BY players DESC LIMIT ?",
				d,
				n,
			).Rows()
			if err != nil {
				logger.Error("Error querying", zap.String("uri", r.URL.RawPath), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			var rval = []struct {
				Name    string `json:"name"`
				Players int    `json:"players"`
			}{}
			defer rows.Close()
			for rows.Next() {
				var row struct {
					Name    string `json:"name"`
					Players int    `json:"players"`
				}
				err := rows.Scan(&row.Name, &row.Players)
				logger.Error("Error scanning", zap.String("uri", r.URL.RawPath), zap.Error(err))
				if err != nil {
					continue
				}
				rval = append(rval, row)
			}
			json.NewEncoder(w).Encode(rval)
		},
	))
}
