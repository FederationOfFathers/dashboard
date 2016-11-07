package api

import (
	"bytes"
	"encoding/json"
	"net/http"
)

var apiRoutes = []string{}
var methodDocs = map[string][]methodDocEntry{}

type methodParams struct {
	Name        string
	Description string
	Type        string
	Values      []string
}
type methodDocEntry struct {
	Method         string
	Description    string
	RequiredParams []methodParams
	OptionalParams []methodParams
	AdminParams    []methodParams
}

func docRouteMethod(route string, method methodDocEntry) {
	if _, ok := methodDocs[route]; !ok {
		apiRoutes = append(apiRoutes, route)
		methodDocs[route] = []methodDocEntry{}
	}
	methodDocs[route] = append(methodDocs[route], method)
}

func docBuild() []byte {
	w := new(bytes.Buffer)
	enc := json.NewEncoder(w)
	w.Write([]byte("{\n"))
	for i, route := range apiRoutes {
		if i > 0 {
			w.Write([]byte(",\n"))
		}
		enc.Encode(route)
		w.Write([]byte(":\n"))
		enc.Encode(methodDocs[route])
	}
	w.Write([]byte("}\n"))
	dec := json.NewDecoder(w)
	var doc interface{}
	dec.Decode(&doc)
	buf, _ := json.MarshalIndent(&doc, "  ", "  ")
	return buf
}

func init() {
	docRouteMethod("/api/v0/login", methodDocEntry{
		Method:      "GET",
		Description: "Logs a user into the API. Linked to from slack messages for users",
	})

	docRouteMethod("/api/v0/logout", methodDocEntry{
		Method:      "GET",
		Description: "Log a user out of the API.",
	})

	docRouteMethod("/api/v0/groups", methodDocEntry{
		Method:      "GET",
		Description: "List all visible groups. List all groups for Admins",
	})

	docRouteMethod("/api/v0/groups/{groupID}/join", methodDocEntry{
		Method:      "GET",
		Description: "Join the group specified by groupID (see /api/v0/groups) if it is marked visible",
	})

	docRouteMethod("/api/v0/groups/{groupID}/visibility", methodDocEntry{
		Method:      "PUT",
		Description: "Set the visibility of the group",
		AdminParams: []methodParams{
			methodParams{
				Name:        "visible",
				Type:        "string",
				Values:      []string{"true", "false"},
				Description: "Whether or not the group should be visible",
			},
		},
	})

	docRouteMethod("/api/v0/ping", methodDocEntry{
		Method:      "GET",
		Description: "Make sure the user is logged in and return information about their current state",
	})

	docRouteMethod("/api/v0/streams", methodDocEntry{
		Method:      "GET",
		Description: "List all registered streams",
	})

	docRouteMethod("/api/v0/streams/{slackID}", methodDocEntry{
		Method:      "GET",
		Description: "Get details for a specific streamer. Probably not used",
	})

	docRouteMethod("/api/v0/streams/{slackID}/{type}", methodDocEntry{
		Method:      "DELETE",
		Description: "Delete a stream if the user is the owner or an admin",
	})

	docRouteMethod("/api/v0/streams", methodDocEntry{
		Method:      "POST",
		Description: "Add a stream",
		RequiredParams: []methodParams{
			methodParams{
				Name:        "kind",
				Type:        "string",
				Values:      []string{"twitch", "youtube"},
				Description: "Which service the stream streams from",
			},
			methodParams{
				Name:        "id",
				Type:        "string",
				Description: "The identifier on the streaming service for the stream",
			},
		},
		AdminParams: []methodParams{
			methodParams{
				Name:        "userID",
				Type:        "string",
				Description: "The slack UserID to assign the stream to",
			},
		},
	})

	docs := docBuild()
	Router.Path("/api/v0/documentation").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write(docs)
		},
	))
}
