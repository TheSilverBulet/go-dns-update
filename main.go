package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// CloudFlare Endpoints
const LIST_ZONES_ENDPOINT = "https://api.cloudflare.com/client/v4/zones"
const LIST_DNS_ENDPOINT = "https://api.cloudflare.com/client/v4/zones/%v/dns_records"

// Other Endpoints
const PUB_IP_SERVICE_ENDPOINT = "https://api.ipify.org"

// Other constants
const AUTH_HEADER_KEY = "Authorization"
const GET_METHOD_KEY = "GET"

// The main function
// Retrieves important values from the CLI flags
func main() {
	// required vars for application run
	var apiToken string
	var logLevel string
	var domainName string
	var handleWWW bool
	// CLI flags for application run
	flag.StringVar(&apiToken, "token", "", "API Token for requests")
	flag.StringVar(&logLevel, "logLevel", "Warn", "Log level to set")
	flag.StringVar(&domainName, "domainName", "", "The domain name to update")
	flag.BoolVar(&handleWWW, "handleWWW", false, "Sometimes a separate www. domain is available for the same host, if this flag is set, it will update both values with the same IP address.")
	flag.Parse()

	// Configure log-level
	switch logLevel {
	case "Info":
		log.SetLevel(log.InfoLevel)
	case "Warn":
		log.SetLevel(log.WarnLevel)
	case "Fatal":
		log.SetLevel(log.FatalLevel)
	case "Error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}

	// Create a context which enables a 5s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	zoneID, err := GetZoneID(*client, ctx, apiToken)
	if err != nil {
		log.Fatal("Error getting Zone ID")
		return
	}
	publicIP, err := GetPublicIP(*client, ctx)
	if err != nil {
		log.Fatal("Error obtaining Public IP Address")
		return
	}
	dnsRecordIP, domainID, wwwID, err := GetDNSRecord(*client, ctx, zoneID, apiToken, domainName, handleWWW)
	if err != nil {
		log.Fatal("Error obtaining DNS Record")
		return
	}
	if publicIP == dnsRecordIP {
		log.Info("A Record IP Address correct, nothing to do")
		return
	}
	updateDNSRecord(*client, ctx, zoneID, apiToken, publicIP, domainID, handleWWW, wwwID)

}

// Method to get Zone Information
// Takes a premade http.Client, context.Context, and a CloudFlare API Token as params
func GetZoneID(client http.Client, ctx context.Context, apiToken string) (string, error) {
	// Create the request setting the context, the method, the endpoint, and the body
	// GET requests don't have a body so pass nil
	req, err := http.NewRequestWithContext(ctx, GET_METHOD_KEY, LIST_ZONES_ENDPOINT, nil)
	if err != nil { // Errors related to creating request
		log.Fatal("Error creating the request")
		return "", err
	}
	// Add the Authorization header to the request using the API Token
	req.Header.Set(AUTH_HEADER_KEY, MakeAuthHeaderValue(apiToken))
	resp, err := client.Do(req) // Fire the request
	if err != nil {             // Errors related to firing the request
		log.Fatal("Error firing request")
		return "", err
	}

	defer resp.Body.Close()            // Make sure to close the response Body
	body, err := io.ReadAll(resp.Body) // Read resp.Body to body var
	if err != nil {                    // Errors related to reading the response
		log.Fatal("Error reading response body")
		return "", err
	}
	var responseResult map[string]any                     // Var of map type key is a string and the value is an object (typically a map)
	json.Unmarshal([]byte(string(body)), &responseResult) // Parse the JSON
	// Parsing the zone ID begins by getting the result JSON array, then getting the first element in the result JSON array which is the account object, then get the value of the id key from the account object
	zoneID := responseResult["result"].([]any)[0].(map[string]any)["id"]
	return zoneID.(string), nil
}

// Method to reach out to the ipify web service and get the value of the running machine's Public IP address
func GetPublicIP(client http.Client, ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, GET_METHOD_KEY, PUB_IP_SERVICE_ENDPOINT, nil)
	if err != nil {
		log.Fatal("Error creating request")
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error firing request")
		return "", err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body) // Read resp.Body to body var
	if err != nil {                    // Errors related to reading the response
		log.Fatal("Error reading response body")
		return "", err
	}

	return string(body), nil
}

// Method to retrieve DNS record information for a provided domainName
// This method returns 3 strings and an error object
// The first string is expected to be the current domain IP address
// The second string is expected to be the ID of the domain record
// The third string is expected to be the ID of the www record (if it exists)
func GetDNSRecord(client http.Client, ctx context.Context, zoneID string, apiToken string, domainName string, handleWWW bool) (string, string, string, error) {
	// Create the request setting the context, the method, the endpoint, and the body
	// GET requests don't have a body so pass nil
	formattedDNSRequestURL := fmt.Sprintf(LIST_DNS_ENDPOINT, zoneID)
	req, err := http.NewRequestWithContext(ctx, GET_METHOD_KEY, formattedDNSRequestURL, nil)
	if err != nil { // Errors related to creating request
		log.Fatal("Error creating the request")
		return "", "", "", err
	}
	// Add the Authorization header to the request using the API Token
	req.Header.Set(AUTH_HEADER_KEY, MakeAuthHeaderValue(apiToken))
	resp, err := client.Do(req) // Fire the request
	if err != nil {             // Errors related to firing the request
		log.Fatal("Error firing request")
		return "", "", "", err
	}

	defer resp.Body.Close()            // Make sure to close the response Body
	body, err := io.ReadAll(resp.Body) // Read resp.Body to body var
	if err != nil {                    // Errors related to reading the response
		log.Fatal("Error reading response body")
		return "", "", "", err
	}
	var responseResult map[string]any                     // Var of map type key is a string and the value is an object (typically a map)
	json.Unmarshal([]byte(string(body)), &responseResult) // Parse the JSON
	jsonResponseArray := responseResult["result"].([]any)
	var domainIP string
	var domainID string
	var wwwID string
	currentElement := 0
	for currentElement < len(jsonResponseArray) {
		tmpJsonObj := jsonResponseArray[currentElement].(map[string]any)
		if tmpJsonObj["name"] == domainName {
			domainIP = tmpJsonObj["content"].(string)
			domainID = tmpJsonObj["id"].(string)
		}
		if handleWWW && tmpJsonObj["name"] == "www."+domainName {
			wwwID = tmpJsonObj["id"].(string)
		}
		currentElement += 1
	}
	if domainIP == "" {
		return "", "", "", fmt.Errorf("could not match domain name")
	}
	if handleWWW && wwwID == "" {
		return "", "", "", fmt.Errorf("could not get www record using domain %v, does it exist?", domainName)
	}
	return domainIP, domainID, wwwID, nil
}

// Method to update the DNS record associated with the provided IDs.
// Returns a boolean value, true if successful, false if not
func updateDNSRecord(client http.Client, ctx context.Context, zoneID string, apiToken string, publicIP string, domainID string, handleWWW bool, wwwID string) (bool, error) {
	return false, nil
}

// Little helper function to help create the value portion of the
// Authorization header for requests
func MakeAuthHeaderValue(apiToken string) string {
	return fmt.Sprintf("Bearer %v", apiToken)
}
