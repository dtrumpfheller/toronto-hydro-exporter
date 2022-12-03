package torontohydro

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dtrumpfheller/toronto-hydro-exporter/helpers"
	"github.com/gocarina/gocsv"
)

type DateTime struct {
	time.Time
}

type ElectricConsumption struct {
	TimeTemp      string    `csv:"Time"`
	UsageOffPeak  float32   `csv:"Usage off-peak (kWh)"`
	UsageMidPeak  float32   `csv:"Usage mid-peak (kWh)"`
	UsageOnPeak   float32   `csv:"Usage on-peak (kWh)"`
	UsageLowTier  float32   `csv:"Usage low-tier (kWh)"`
	UsageHighTier float32   `csv:"Usage high-tier (kWh)"`
	CostOffPeak   float32   `csv:"Cost off-peak ($)"`
	CostMidPeak   float32   `csv:"Cost mid-peak ($)"`
	CostOnPeak    float32   `csv:"Cost on-peak ($)"`
	CostLowTier   float32   `csv:"Cost low-tier ($)"`
	CostHighTier  float32   `csv:"Cost high-tier ($)"`
	Time          time.Time `csv:"-"`
}

var client http.Client

func Login(config helpers.Config) error {

	log.Println("Logging into Toronto Hydro... ")

	// create cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Printf("Got error while creating cookie jar [%s]!", err.Error())
	}
	client = http.Client{
		Jar: jar,
	}

	// get login page
	url := "https://www.torontohydro.com/log-in"
	if config.TorontoHydro.Mock {
		url = "http://localhost:9999/log-in"
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error getting Toronto Hydro login page [%s]!\n", err.Error())
		return err
	}
	if resp.StatusCode != 200 {
		log.Printf("Getting Toronto Hydro login page failed with status code [%d]!\n", resp.StatusCode)
		return errors.New("Error")
	}

	// extract login portlet url
	defer resp.Body.Close()
	loginPageBody, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Error processing Toronto Hydro login page [%s]!\n", err.Error())
		return err
	}
	loginUrl := loginPageBody.Find("#_th_module_authentication_ThModuleAuthenticationPortlet_authentication").AttrOr("action", "")
	if config.TorontoHydro.Mock {
		loginUrl = "http://localhost:9999/log-in"
	}

	// logging in
	body := "_th_module_authentication_ThModuleAuthenticationPortlet_email=" + config.TorontoHydro.Username + "&_th_module_authentication_ThModuleAuthenticationPortlet_password=" + config.TorontoHydro.Password
	req, err = http.NewRequest("POST", loginUrl, bytes.NewBufferString(body))
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err = client.Do(req)
	if err != nil {
		log.Printf("Error logging into Toronto Hydro [%s]!\n", err.Error())
		return err
	}
	if resp.StatusCode != 200 {
		log.Printf("Logging into Toronto Hydro failed with status code [%d]!\n", resp.StatusCode)
		return errors.New("Error")
	}

	return nil
}

func Logout(config helpers.Config) error {

	log.Println("Logging out of Toronto Hydro... ")

	url := "https://www.torontohydro.com/c/portal/logout"
	if config.TorontoHydro.Mock {
		url = "http://localhost:9999/logout"
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error logging out [%s]!\n", err.Error())
		return err
	}
	if resp.StatusCode != 200 {
		log.Printf("Logging out failed with status code [%d]!\n", resp.StatusCode)
		return errors.New("Error")
	}

	return nil
}

func GetData(date time.Time, config helpers.Config) ([]*ElectricConsumption, error) {

	dateString := date.Format("2006-01-02")
	log.Println("Getting consumption data for " + dateString)

	// get data
	url := "https://www.torontohydro.com/my-account/my-usage?p_p_id=thmoduletou&p_p_lifecycle=2&p_p_state=normal&p_p_mode=view&p_p_resource_id=getHourlyChartData&p_p_cacheability=cacheLevelPage"
	if config.TorontoHydro.Mock {
		url = "http://localhost:9999/my-usage"
	}
	body := "spIDs=" + config.TorontoHydro.Id + "&meterNum=" + config.TorontoHydro.Meter + "&date=" + dateString
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	if err != nil {
		log.Fatalf("Got error %s", err.Error())
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error getting data from Toronto Hydro [%s]!\n", err.Error())
		return nil, err
	}
	if resp.StatusCode != 200 {
		log.Printf("Calling Toronto Hydro failed with status code [%d]!\n", resp.StatusCode)
		return nil, errors.New("Error")
	}
	defer resp.Body.Close()
	dataBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error processing response from Toronto Hydro [%s]!\n", err.Error())
		return nil, err
	}

	// remove comments
	scanner := bufio.NewScanner(strings.NewReader(string(dataBody)))
	filteredData := ""
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "#") {
			filteredData += line + "\n"
		}
	}

	// read data
	consumptions := []*ElectricConsumption{}
	err = gocsv.UnmarshalString(filteredData, &consumptions)
	if err != nil {
		log.Printf("Error processing response from Toronto Hydro [%s]!\n", err.Error())
		return nil, err
	}

	// cleanup
	loc, err := time.LoadLocation("America/Toronto")
	if err != nil {
		log.Printf("Error getting current time location [%s]!\n", err.Error())
		return nil, err
	}

	for _, consumption := range consumptions {
		dateTime := getDateTime(consumption.TimeTemp, date, loc)
		consumption.Time = dateTime
	}

	return consumptions, nil
}

func getDateTime(value string, date time.Time, location *time.Location) time.Time {
	// could not figure out how to do this better...
	switch {
	case value == "12  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location)
	case value == "1  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 1, 0, 0, 0, location)
	case value == "2  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 2, 0, 0, 0, location)
	case value == "3  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 3, 0, 0, 0, location)
	case value == "4  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 4, 0, 0, 0, location)
	case value == "5  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 5, 0, 0, 0, location)
	case value == "6  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 6, 0, 0, 0, location)
	case value == "7  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 7, 0, 0, 0, location)
	case value == "8  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 8, 0, 0, 0, location)
	case value == "9  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, location)
	case value == "10  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 10, 0, 0, 0, location)
	case value == "11  a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 11, 0, 0, 0, location)
	case value == "12  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, location)
	case value == "1  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 13, 0, 0, 0, location)
	case value == "2  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 14, 0, 0, 0, location)
	case value == "3  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 15, 0, 0, 0, location)
	case value == "4  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 16, 0, 0, 0, location)
	case value == "5  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, location)
	case value == "6  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 18, 0, 0, 0, location)
	case value == "7  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 19, 0, 0, 0, location)
	case value == "8  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 20, 0, 0, 0, location)
	case value == "9  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 21, 0, 0, 0, location)
	case value == "10  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 22, 0, 0, 0, location)
	case value == "11  p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 23, 0, 0, 0, location)
	default:
		return time.Now()
	}
}
