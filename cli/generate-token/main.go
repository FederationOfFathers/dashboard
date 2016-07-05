package main

import (
	"flag"
	"fmt"

	"github.com/FederationOfFathers/dashboard/api"
	"github.com/apokalyptik/cfg"
)

var who string

func main() {
	flag.StringVar(&who, "who", who, "for who?")
	acfg := cfg.New("cfg-api")
	acfg.StringVar(&api.AuthSecret, "secret", api.AuthSecret, "Authentication secret for use in generating login tokens")
	cfg.Parse()
	fmt.Println(api.GenerateValidAuthTokens(who)[0])
}
