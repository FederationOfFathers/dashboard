package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

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
			w.Header().Set("Content-Type", "text/json")
			days, _ := strconv.Atoi(mux.Vars(r)["days"])
			user := mux.Vars(r)["id"]
			rows, err := DB.Raw(
				strings.Join([]string{
					"SELECT g.id, g.platform, g.platform_id, g.name, g.image, mg.played",
					"FROM membergames mg",
					"JOIN games g ON (mg.game = g.id)",
					"JOIN members m ON (mg.member = m.id)",
					"WHERE m.slack = ?",
					"AND mg.played >= DATE_SUB(NOW(), INTERVAL ? DAY)",
				}, " "),
				user,
				days,
			).Rows()
			if err != nil {
				Logger.Error("querying player games", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			var rval = []struct {
				ID         int       `json:"id"`
				Platform   int       `json:"platform"`
				PlatformID int       `json:"platform_id"`
				Name       string    `json:"name"`
				Image      string    `json:"image"`
				Played     time.Time `json:"played"`
			}{}
			for rows.Next() {
				var row = struct {
					ID         int       `json:"id"`
					Platform   int       `json:"platform"`
					PlatformID int       `json:"platform_id"`
					Name       string    `json:"name"`
					Image      string    `json:"image"`
					Played     time.Time `json:"played"`
				}{}
				err := rows.Scan(
					&row.ID,
					&row.Platform,
					&row.PlatformID,
					&row.Name,
					&row.Image,
					&row.Played,
				)
				if err != nil {
					Logger.Error("Error scanning", zap.Error(err))
					continue
				}
				rval = append(rval, row)
			}
			json.NewEncoder(w).Encode(rval)
		}))
	Router.Path("/api/v0/games/played/{game}/{days}.json").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			id, _ := strconv.Atoi(mux.Vars(r)["game"])
			var game = struct {
				ID         int    `json:"id"`
				Name       string `json:"name"`
				Image      string `json:"image"`
				Platform   int    `json:"platform"`
				PlatformID int    `json:"platform_id"`
			}{}
			err := DB.Raw("SELECT id,name,image,platform,platform_id FROM games WHERE id=? LIMIT 1", id).Row().Scan(
				&game.ID,
				&game.Name,
				&game.Image,
				&game.Platform,
				&game.PlatformID,
			)
			if err != nil {
				Logger.Error("eror querying game", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			d, _ := strconv.Atoi(mux.Vars(r)["days"])
			rows, err := DB.Raw(strings.Join([]string{
				"SELECT slack,played",
				"FROM members m",
				"JOIN membergames mg ON (mg.member=m.id)",
				"WHERE mg.game = ?",
				"AND m.seen > (UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL 30 DAY)))",
				"AND mg.played > DATE_SUB(NOW(), INTERVAL ? DAY)",
				"ORDER BY played DESC",
			}, " "),
				id,
				d,
			).Rows()
			if err != nil {
				Logger.Error("Error querying", zap.String("uri", r.URL.RawPath), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			var rval = struct {
				Game struct {
					ID         int    `json:"id"`
					Name       string `json:"name"`
					Image      string `json:"image"`
					Platform   int    `json:"platform"`
					PlatformID int    `json:"platform_id"`
				} `json:"game"`
				Players []struct {
					Slack  string    `json:"slack_id"`
					Played time.Time `json:"played"`
				} `json:"players"`
			}{
				Game: game,
			}
			defer rows.Close()
			for rows.Next() {
				var row struct {
					Slack  string    `json:"slack_id"`
					Played time.Time `json:"played"`
				}
				err := rows.Scan(&row.Slack, &row.Played)
				if err != nil {
					Logger.Error("Error scanning", zap.String("uri", r.URL.RawPath), zap.Error(err))
					continue
				}
				rval.Players = append(rval.Players, row)
			}
			json.NewEncoder(w).Encode(rval)
		}))
	Router.Path("/api/v0/games/played/top/{days}/{number}.json").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			n, _ := strconv.Atoi(mux.Vars(r)["number"])
			d, _ := strconv.Atoi(mux.Vars(r)["days"])
			rows, err := DB.Raw(
				"SELECT g.id, g.name, COUNT(mg.member) as players "+
					"FROM membergames mg JOIN games g ON( mg.game = g.id ) JOIN members m ON( mg.member = m.id) "+
					"WHERE mg.played > DATE_SUB(NOW(), INTERVAL ? DAY) "+
					"AND m.seen > (UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL 30 DAY))) "+
					"GROUP BY g.id ORDER BY players DESC LIMIT ?",
				d,
				n,
			).Rows()
			if err != nil {
				Logger.Error("Error querying", zap.String("uri", r.URL.RawPath), zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			var rval = []struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Image   string `json:"image"`
				Players int    `json:"players"`
			}{}
			defer rows.Close()
			for rows.Next() {
				var row struct {
					ID      int    `json:"id"`
					Name    string `json:"name"`
					Image   string `json:"image"`
					Players int    `json:"players"`
				}
				err := rows.Scan(&row.ID, &row.Name, &row.Players)
				if err != nil {
					Logger.Error("Error scanning", zap.String("uri", r.URL.RawPath), zap.Error(err))
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
