package influxdb

import (
	"container/list"
	"context"
	"log"
	"strconv"
	"time"

	"github.com/dtrumpfheller/toronto-hydro-exporter/helpers"
	"github.com/dtrumpfheller/toronto-hydro-exporter/torontohydro"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

func Export(meter torontohydro.Meter, consumptions *list.List, config helpers.Config) {

	// create client objects
	client := influxdb2.NewClient(config.InfluxDB.URL, config.InfluxDB.Token)
	queryAPI := client.QueryAPI(config.InfluxDB.Organization)
	writeAPI := client.WriteAPI(config.InfluxDB.Organization, config.InfluxDB.Bucket)

	// start & end can be determined based on list elements
	startDateTime := consumptions.Front().Value.(*torontohydro.ElectricConsumption).Time.Add(-1 * time.Hour)
	endDateTime := consumptions.Back().Value.(*torontohydro.ElectricConsumption).Time.Add(1 * time.Hour)

	// check if entry is already stored, only consider last 7 days
	query := `from(bucket: "` + config.InfluxDB.Bucket + `")
		|> range(start: ` + strconv.FormatInt(startDateTime.Unix(), 10) + `, stop: ` + strconv.FormatInt(endDateTime.Unix(), 10) + `)
		|> filter(fn: (r) => r["_measurement"] == "toronto_hydro")
		|> filter(fn: (r) => r["meter"] == "` + meter.MeterNumber + `")
		|> filter(fn: (r) => r["_field"] == "UsageHighTier" or r["_field"] == "UsageLowTier" or r["_field"] == "UsageMidPeak" or r["_field"] == "UsageOffPeak" or r["_field"] == "UsageOnPeak")`
	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		log.Printf("Error calling InfluxDB [%s]!\n", err.Error())
		return
	}

	// remove consumptions that have already been submitted
	for result.Next() {
		var next *list.Element
		for e := consumptions.Front(); e != nil; e = next {
			next = e.Next()
			if e.Value.(*torontohydro.ElectricConsumption).Time.Equal(result.Record().Time()) {
				consumptions.Remove(e)
				break
			}
		}
	}

	if consumptions.Len() > 0 {
		// write remaining consumptions to influxdb
		for e := consumptions.Front(); e != nil; e = e.Next() {
			consumption := e.Value.(*torontohydro.ElectricConsumption)
			if !hasData(consumption) {
				log.Println("No data for " + consumption.Time.Format("2006-01-02 15:04:05"))
				continue
			}
			log.Println("Inserting " + consumption.Time.Format("2006-01-02 15:04:05"))
			point := influxdb2.NewPointWithMeasurement("toronto_hydro").
				AddTag("meter", meter.MeterNumber).
				SetTime(consumption.Time)
			addField("UsageHighTier", consumption.UsageHighTier, point)
			addField("UsageLowTier", consumption.UsageLowTier, point)
			addField("UsageOnPeak", consumption.UsageOnPeak, point)
			addField("UsageMidPeak", consumption.UsageMidPeak, point)
			addField("UsageOffPeak", consumption.UsageOffPeak, point)
			addField("CostHighTier", consumption.CostHighTier, point)
			addField("CostLowTier", consumption.CostLowTier, point)
			addField("CostOnPeak", consumption.CostOnPeak, point)
			addField("CostMidPeak", consumption.CostMidPeak, point)
			addField("CostOffPeak", consumption.CostOffPeak, point)
			writeAPI.WritePoint(point)
		}

		// force all unwritten data to be sent
		writeAPI.Flush()

	} else {
		log.Println("No new metrics available, skip export to influx")
	}

	// ensures background processes finishes
	client.Close()
}

func addField(name string, value float32, point *write.Point) {
	if value > 0.0 {
		point.AddField(name, value)
	}
}

func hasData(consumption *torontohydro.ElectricConsumption) bool {
	return consumption.UsageHighTier > 0.0 ||
		consumption.UsageLowTier > 0.0 ||
		consumption.UsageOnPeak > 0.0 ||
		consumption.UsageMidPeak > 0.0 ||
		consumption.UsageOffPeak > 0.0 ||
		consumption.CostHighTier > 0.0 ||
		consumption.CostLowTier > 0.0 ||
		consumption.CostOnPeak > 0.0 ||
		consumption.CostMidPeak > 0.0 ||
		consumption.CostOffPeak > 0.0
}
