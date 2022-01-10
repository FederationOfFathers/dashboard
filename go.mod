module github.com/FederationOfFathers/dashboard

go 1.16

require (
	github.com/apokalyptik/cfg v0.0.0-20160401174707-703f89116901
	github.com/bearcherian/rollzap v1.0.2
	github.com/boltdb/bolt v1.3.1
	github.com/bwmarrin/discordgo v0.23.3-0.20210529215543-f5bb723db8d9
	github.com/denisenkom/go-mssqldb v0.0.0-20190915052044-aa4949efa320 // indirect
	github.com/dineshappavoo/basex v0.0.0-20160618072718-f35bafba529c
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/securecookie v0.0.0-20160422134519-667fe4e3466a
	github.com/honeycombio/beeline-go v1.1.2
	github.com/jinzhu/gorm v0.0.0-20160404144928-5174cc5c242a
	github.com/jinzhu/inflection v0.0.0-20170102125226-1c35d901db3d // indirect
	github.com/jinzhu/now v1.0.1 // indirect
	github.com/lusis/slack-test v0.0.0-20190426140909-c40012f20018 // indirect
	github.com/nicklaw5/helix v1.17.0
	github.com/nlopes/slack v0.0.0-20180905213137-8cf10c586222
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c
	github.com/robfig/cron v1.2.0
	github.com/rollbar/rollbar-go v1.4.0
	github.com/speps/go-hashids v2.0.0+incompatible
	go.uber.org/zap v1.17.0
	golang.org/x/oauth2 v0.0.0-20190130055435-99b60b757ec1
	gopkg.in/djherbis/stow.v2 v2.2.0
	gopkg.in/yaml.v2 v2.4.0
)

// this is a fork that has an unreleased version of interaction component implementation
replace github.com/bwmarrin/discordgo v0.23.3-0.20210529215543-f5bb723db8d9 => github.com/FedorLap2006/discordgo v0.22.1-0.20210618185457-afb10575dbd8
