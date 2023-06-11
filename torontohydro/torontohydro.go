package torontohydro

import (
	"bufio"
	"bytes"
	"encoding/json"
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
	TimeTemp          string    `csv:"Time"`
	UsageTOUOffPeak   float32   `csv:"Usage TOU off-peak (kWh)"`
	UsageTOUMidPeak   float32   `csv:"Usage TOU mid-peak (kWh)"`
	UsageTOUOnPeak    float32   `csv:"Usage TOU on-peak (kWh)"`
	UsageLowTier      float32   `csv:"Usage tier 1 (kWh)"`
	UsageHighTier     float32   `csv:"Usage tier 2 (kWh)"`
	UsageULOOvernight float32   `csv:"Usage ULO overnight (kWh)"`
	UsageULOOffPeal   float32   `csv:"Usage ULO off-peak (kWh)"`
	UsageULOMidPeak   float32   `csv:"Usage ULO mid-peak (kWh)"`
	UsageULOOnPeak    float32   `csv:"Usage ULO on-peak (kWh)"`
	CostTOUOffPeak    float32   `csv:"Cost TOU off-peak ($)"`
	CostTOUMidPeak    float32   `csv:"Cost TOU mid-peak ($)"`
	CostTOUOnPeak     float32   `csv:"Cost TOU on-peak ($)"`
	CostLowTier       float32   `csv:"Cost tier 1 ($)"`
	CostHighTier      float32   `csv:"Cost tier 2 ($)"`
	CostULOOvernight  float32   `csv:"Cost ULO overnight ($)"`
	CostULOOffPeal    float32   `csv:"Cost ULO off-peak ($)"`
	CostULOMidPeak    float32   `csv:"Cost ULO mid-peak ($)"`
	CostULOOnPeak     float32   `csv:"Cost ULO on-peak ($)"`
	Time              time.Time `csv:"-"`
}

type Meter struct {
	MeterNumber string `json:"meterNum"`
	Id          string `json:"id"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
}

var client http.Client

func Login(config helpers.Config) error {

	log.Println("Logging into Toronto Hydro... ")

	// create cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Printf("Got error while creating cookie jar [%s]!", err.Error())
		return err
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
		log.Printf("Got error %s", err.Error())
		return err
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
		log.Printf("Got error %s", err.Error())
		return err
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

func GetMeters(config helpers.Config) ([]Meter, error) {

	log.Println("Getting meter list")

	// get data
	url := "https://www.torontohydro.com/my-account/my-usage?p_p_id=thmoduletou&p_p_lifecycle=2&p_p_state=normal&p_p_mode=view&p_p_resource_id=fetchMeterList&p_p_cacheability=cacheLevelPage"
	if config.TorontoHydro.Mock {
		url = "http://localhost:9999/my-usage?p_p_resource_id=fetchMeterList"
	}
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Printf("Got error %s", err.Error())
		return nil, err
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

	// extract body
	defer resp.Body.Close()
	var meters []Meter
	err = json.NewDecoder(resp.Body).Decode(&meters)
	if err != nil {
		log.Printf("Error processing Toronto Water account details response [%s]!\n", err.Error())
		return nil, err
	}

	return meters, nil
}

func GetData(meter Meter, date time.Time, config helpers.Config) ([]*ElectricConsumption, error) {

	dateString := date.Format("2006-01-02")
	log.Println("Getting consumption data for meter " + meter.MeterNumber + " and date " + dateString)

	// get data
	url := "https://www.torontohydro.com/my-account/my-usage?p_p_id=thmoduletou&p_p_lifecycle=2&p_p_state=normal&p_p_mode=view&p_p_resource_id=getHourlyChartData&p_p_cacheability=cacheLevelPage"
	if config.TorontoHydro.Mock {
		url = "http://localhost:9999/my-usage?p_p_resource_id=getHourlyChartData"
	}
	body := "spIDs=" + meter.Id + "&meterNum=" + meter.MeterNumber + "&date=" + dateString
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	if err != nil {
		log.Printf("Got error %s", err.Error())
		return nil, err
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
	for _, consumption := range consumptions {
		dateTime := getDateTime(consumption.TimeTemp, date)
		consumption.Time = dateTime
	}

	return consumptions, nil
}

func getDateTime(value string, date time.Time) time.Time {
	// could not figure out how to do this better...
	switch {
	case value == "12 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	case value == "1 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 1, 0, 0, 0, date.Location())
	case value == "2 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 2, 0, 0, 0, date.Location())
	case value == "3 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 3, 0, 0, 0, date.Location())
	case value == "4 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 4, 0, 0, 0, date.Location())
	case value == "5 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 5, 0, 0, 0, date.Location())
	case value == "6 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 6, 0, 0, 0, date.Location())
	case value == "7 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 7, 0, 0, 0, date.Location())
	case value == "8 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 8, 0, 0, 0, date.Location())
	case value == "9 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, date.Location())
	case value == "10 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 10, 0, 0, 0, date.Location())
	case value == "11 a.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 11, 0, 0, 0, date.Location())
	case value == "12 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, date.Location())
	case value == "1 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 13, 0, 0, 0, date.Location())
	case value == "2 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 14, 0, 0, 0, date.Location())
	case value == "3 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 15, 0, 0, 0, date.Location())
	case value == "4 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 16, 0, 0, 0, date.Location())
	case value == "5 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, date.Location())
	case value == "6 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 18, 0, 0, 0, date.Location())
	case value == "7 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 19, 0, 0, 0, date.Location())
	case value == "8 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 20, 0, 0, 0, date.Location())
	case value == "9 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 21, 0, 0, 0, date.Location())
	case value == "10 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 22, 0, 0, 0, date.Location())
	case value == "11 p.m.":
		return time.Date(date.Year(), date.Month(), date.Day(), 23, 0, 0, 0, date.Location())
	default:
		log.Printf("Error determining date [%s]!\n", value)
		return time.Now()
	}
}
