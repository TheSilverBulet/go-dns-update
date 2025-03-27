package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
	log "github.com/sirupsen/logrus"
)

// Other Endpoints
const PUB_IP_SERVICE_ENDPOINT = "https://api.ipify.org"

// HTTP Method Constants
const GET_METHOD_KEY = "GET"

func main() {
	// required vars for application run
	var apiToken string
	var logLevel string
	var domainName string
	var handleWWW bool
	// CLI flags for application run
	flag.StringVar(&apiToken, "token", "", "Required. API Token for requests.")
	flag.StringVar(&logLevel, "logLevel", "Warn", "Log level to set. Defaults to Warn.")
	flag.StringVar(&domainName, "domainName", "", "Required. The domain name to update.")
	flag.BoolVar(&handleWWW, "handleWWW", false, "Sometimes a separate www domain is available for the same root domain name, if this flag is set, it will update both the root domain name and the www domain name values with the same IP address. Defaults to false.")
	flag.Parse()

	// Configure log-level
	SetLogLevel(logLevel)

	// No point in continuing execution if these flags are not provided
	if apiToken == "" || domainName == "" {
		log.Fatal("No values provided for apiToken flag, nor domainName flag. Aborting...")
		return
	}

	// create Cloudflare client
	// pass in the provided api token
	// set the request timeout to 5 seconds
	// the default retry amount is 2
	cfClient := cloudflare.NewClient(
		option.WithAPIToken(apiToken),
		option.WithRequestTimeout(5*time.Second),
	)

	// Get Zone ID
	zoneID, err := GetZoneID(*cfClient, domainName)
	if err != nil {
		log.Fatal("Could not retrieve Zone ID")
		return
	}

	// Get current Public IP
	publicIP, err := GetPublicIP()
	if err != nil {
		log.Fatal("Error obtaining Public IP Address")
		return
	}

	// Get DNS Records
	domainID, domainIP, wwwDomainID, err := GetDNSRecords(*cfClient, domainName, zoneID, handleWWW)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	// If for some reason this comes back blank, fail
	if domainID == "" {
		log.Fatal("Couldn't obtain A Record ID")
		return
	}
	// If for some reason this comes back blank, fail
	if handleWWW && wwwDomainID == "" {
		log.Fatal(`Couldn't obtain 'www' A Record ID`)
		return
	}

	// If the publicly obtained IP matches our current DNS A Record IP, all set
	if publicIP == domainIP {
		// Straight up print this line to console so we can see that it is effectively doing something without increasing log granularity
		fmt.Println(`DNS Record IP Address matches external IP address, nothing to do`)
		return
	}

	// Only ends up here in the event that the DNS Records needs to be updated
	err = UpdateDNSRecord(*cfClient, domainName, zoneID, publicIP, domainID, wwwDomainID, handleWWW)
	if err != nil {
		log.Fatal("Failed to update DNS Records")
	}

}

// Helper method to get the Zone ID associated with the provided API Token
func GetZoneID(cfClient cloudflare.Client, domainName string) (string, error) {
	// Get the zone information associated with the provided API Token
	zone, err := cfClient.Zones.List(context.Background(), zones.ZoneListParams{
		Name: cloudflare.String(domainName),
	})
	if err != nil {
		log.Fatal("Error retrieving zone data")
		return "", err
	}
	// Could be multiple Zones associated to this one token so make sure we are dealing with the one that matches our domain name
	for i := range zone.Result {
		item := zone.Result[i]
		if item.Name == domainName {
			return item.ID, nil
		}
	}
	return "", fmt.Errorf("could not match a Zone ID to the provided domain name")
}

// Method to reach out to the ipify web service and get the value of the running machine's Public IP address
func GetPublicIP() (string, error) {
	// Create a context which enables a 5s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
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

// Helper method to get the current DNS Record information
// return expects this order: domainID, domainIP, wwwDomainID, error
func GetDNSRecords(cfClient cloudflare.Client, domainName string, zoneID string, handleWWW bool) (string, string, string, error) {
	// Get the list of DNS records associated with this Zone ID
	dnsRecordList, err := cfClient.DNS.Records.List(context.Background(), dns.RecordListParams{
		ZoneID: cloudflare.String(zoneID),
	})
	if err != nil {
		return "", "", "", err
	}
	var domainID string
	var domainIP string
	var wwwDomainID string
	// For every returned record see which one's 'Name' member matches our domainName, grab the ID and the Content of that record
	// If handling www record, look for the record whose 'Name' member matches our domainName with 'www.' prepended and store that ID
	for i := range dnsRecordList.Result {
		if dnsRecordList.Result[i].Name == domainName {
			domainID = dnsRecordList.Result[i].ID
			domainIP = dnsRecordList.Result[i].Content
		}
		if handleWWW && dnsRecordList.Result[i].Name == fmt.Sprintf("www.%v", domainName) {
			wwwDomainID = dnsRecordList.Result[i].ID
		}
	}
	// Once searching is complete return what we have
	return domainID, domainIP, wwwDomainID, nil
}

func UpdateDNSRecord(cfClient cloudflare.Client, domainName string, zoneID string, publicIP string, domainID string, wwwDomainID string, handleWWW bool) error {
	message, err := cfClient.DNS.Records.Edit(context.Background(), domainID, dns.RecordEditParams{
		ZoneID: cloudflare.String(zoneID),
		Record: dns.ARecordParam{Content: cloudflare.String(publicIP)},
	})
	if err != nil {
		log.Fatal("Error updating A Record")
		return err
	}
	if message.Content == publicIP {
		log.Info(`Main domain A record updated successfully`)
	}
	if handleWWW {
		wwwMessage, err := cfClient.DNS.Records.Edit(context.Background(), domainID, dns.RecordEditParams{
			ZoneID: cloudflare.String(wwwDomainID),
			Record: dns.ARecordParam{Content: cloudflare.String(publicIP)},
		})
		if err != nil {
			log.Fatal("Error updating www A Record")
			return err
		}
		if wwwMessage.Content == publicIP {
			log.Info("www domain A record updated successfully")
		}
	}
	return nil
}

// Helper method to set the log level for the program, defaults to Warn
func SetLogLevel(logLevel string) {
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
}
