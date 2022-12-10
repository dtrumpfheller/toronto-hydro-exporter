package main

import (
	"container/list"
	"flag"
	"log"
	"time"

	"github.com/dtrumpfheller/toronto-hydro-exporter/helpers"
	"github.com/dtrumpfheller/toronto-hydro-exporter/influxdb"
	"github.com/dtrumpfheller/toronto-hydro-exporter/torontohydro"
)

var (
	configFile = flag.String("config", "config.yml", "configuration file")
	config     helpers.Config
)

func main() {

	// load arguments into variables
	flag.Parse()

	// load config file
	config = helpers.ReadConfig(*configFile)

	// setup mock if necessary
	if config.TorontoHydro.Mock {
		torontohydro.Mock()
	}

	for {
		// export metrics
		exportMetrics()

		if config.SleepDuration <= 0 {
			break
		}
		time.Sleep(time.Duration(config.SleepDuration) * time.Minute)
	}
}

func exportMetrics() {
	log.Println("Getting Toronto Hydro energy consumption... ")
	start := time.Now()

	err := torontohydro.Login(config)
	if err != nil {
		return
	}

	meters, err := torontohydro.GetMeters(config)
	if err != nil {
		return
	}

	for _, meter := range meters {
		endDate, _ := time.ParseInLocation("2006-01-02", meter.EndDate, start.Location())
		startDate, _ := time.ParseInLocation("2006-01-02", meter.StartDate, start.Location())

		date := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location()).AddDate(0, 0, -config.LookDaysInPast)
		if date.Before(startDate) {
			date = startDate
		}

		// 1. get data
		consumptions := list.New()

		// for all days until endDate (excluding endDate as it never has values)
		for ok := endDate.After(date); ok; ok = endDate.After(date) {
			data, err := torontohydro.GetData(meter, date, config)
			if err == nil {
				for _, consumption := range data {
					consumptions.PushBack(consumption)
				}
			}
			date = date.AddDate(0, 0, 1)
		}

		// 2. export data
		if consumptions.Len() > 0 {
			influxdb.Export(meter, consumptions, config)
		} else {
			log.Println("No data gathered, skipping export to influxDB")
		}
	}

	torontohydro.Logout(config)

	log.Printf("Finished in %s\n", time.Since(start))
}
