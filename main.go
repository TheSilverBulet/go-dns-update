package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Endpoints
const ZONES_ENDPOINT = "https://api.cloudflare.com/client/v4/zones"

// Other constants
const AUTH_HEADER_KEY = "Authorization"

func main() {
	apiToken := os.Args[1]
	apiEmail := os.Args[2]
	client := &http.Client{}

	GetZoneInformation(*client, apiToken, apiEmail)
}

func GetZoneInformation(client http.Client, apiToken string, apiEmail string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", ZONES_ENDPOINT, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set(AUTH_HEADER_KEY, MakeAuthHeaderValue(apiToken))

	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	fmt.Println(resp)
}

func MakeAuthHeaderValue(apiToken string) string {
	return fmt.Sprintf("Bearer %v", apiToken)
}
