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
	if err == nil {
		date := time.Now().AddDate(0, 0, -config.LookDaysInPast)
		yesterday := time.Now().AddDate(0, 0, -1)

		// 1. get data
		consumptions := list.New()

		// for all days until and including yesterday
		for ok := true; ok; ok = yesterday.After(date) {
			data, err := torontohydro.GetData(date, config)
			if err == nil {
				for _, consumption := range data {
					consumptions.PushBack(consumption)
				}
			}
			date = date.AddDate(0, 0, 1)

			// sleep a bit to not floot any http endpoints
			time.Sleep(500 * time.Millisecond)
		}

		torontohydro.Logout(config)

		// 2. export data
		if consumptions.Len() > 0 {
			influxdb.Export(consumptions, config)
		} else {
			log.Println("No data gathered, skipping export to influxDB")
		}
	}

	log.Printf("Finished in %s\n", time.Since(start))
}
