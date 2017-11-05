package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type gamePlatform int

func (g gamePlatform) name() string {
	if s, ok := map[int]string{
		1: "Xbox360",
		2: "XboxOne",
		3: "PC",
		4: "iOS",
		5: "Android",
		6: "Mobile",
		7: "GearVR",
		8: "Kindle",
	}[int(g)]; ok {
		return s
	}
	return "Unknown"
}

func (g gamePlatform) MarshalJSON() ([]byte, error) {
	return []byte(g.name()), nil
}

func getPicforGameName(name string) string {
	var rval string
	var p int
	rows, err := DB.Raw("SELECT platform,image FROM games WHERE name=?", name).Rows()
	defer rows.Close()
	if err != nil {
		return rval
	}
	for rows.Next() {
		var platform int
		var image string
		if err := rows.Scan(&platform, &image); err != nil {
			continue
		}
		if image == "" {
			continue
		}
		if platform == 2 {
			return image
		}
		if p == 0 || p > platform {
			p = platform
			rval = image
		}
	}
	return rval
}

func init() {
	Router.Path("/api/v0/games/player/{id}/{days}.json").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
		}))
	Router.Path("/api/v0/games/played/{game}/{days}.json").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
		}))
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
				Image   string `json:"image"`
				Players int    `json:"players"`
			}{}
			defer rows.Close()
			for rows.Next() {
				var row struct {
					Name    string `json:"name"`
					Image   string `json:"image"`
					Players int    `json:"players"`
				}
				err := rows.Scan(&row.Name, &row.Players)
				if err != nil {
					logger.Error("Error scanning", zap.String("uri", r.URL.RawPath), zap.Error(err))
					continue
				}
				if image := getPicforGameName(row.Name); image != "" {
					row.Image = "/api/v0/cdn/" + image
				}
				rval = append(rval, row)
			}
			json.NewEncoder(w).Encode(rval)
		},
	))
}
