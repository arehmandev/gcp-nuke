package gcp

import (
	"context"
	"log"
	"net/http"

	"golang.org/x/oauth2/google"
)

// Ctx = context
var Ctx = context.Background()
var clientMap = make(map[string]*http.Client)

func clientFactory(scope string) (client *http.Client) {
	client, ok := clientMap[scope]
	if !ok {
		client, err := google.DefaultClient(Ctx, scope)
		if err != nil {
			log.Fatal(err)
		}
		clientMap[scope] = client
	}

	return clientMap[scope]
}
