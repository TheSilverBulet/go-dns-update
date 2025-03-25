package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// Endpoints
const ZONES_ENDPOINT = "https://api.cloudflare.com/client/v4/zones"

// Other constants
const AUTH_HEADER_KEY = "Authorization"

func main() {
	apiToken := os.Args[1]
	apiEmail := os.Args[2]
	log.Info(apiEmail)
	// Create a context which enables a 5s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	GetZoneInformation(*client, ctx, apiToken)
}

// Method to get Zone Information
func GetZoneInformation(client http.Client, ctx context.Context, apiToken string) {
	// Create the request setting the context, the method, the endpoint, and the body
	// GET requests don't have a body so pass nil
	req, err := http.NewRequestWithContext(ctx, "GET", ZONES_ENDPOINT, nil)

	if err != nil { // Handle any errors relating to creating the request
		log.Panic("Error creating the request")
	}
	// Add the Authorization header to the request using the API Token
	req.Header.Set(AUTH_HEADER_KEY, MakeAuthHeaderValue(apiToken))
	resp, err := client.Do(req) // Fire the request

	if err != nil {
		log.Panic("Error firing request")
	}

	defer resp.Body.Close()            // Make sure to close the response Body
	body, err := io.ReadAll(resp.Body) // Easier way to read the response body rather than manually managing a new byte[]
	if err != nil {
		log.Panic("Error reading response body")
	}
	//Cast the byte[] as a string to read the body JSON normally
	log.Info(string(body))
}

func MakeAuthHeaderValue(apiToken string) string {
	return fmt.Sprintf("Bearer %v", apiToken)
}
