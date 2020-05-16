package influxdb

import (
	"fmt"
	"time"

	_ "github.com/influxdata/influxdb1-client" // this is important because of a bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/pdunnavant/modem-scraper/config"
	"github.com/pdunnavant/modem-scraper/scrape"
)

// Publish publishes the data within modemInformation to
// the InfluxDB server configuration within the given
// configuration.
func Publish(config config.InfluxDB, modemInformation scrape.ModemInformation) error {
	start := time.Now()

	fmt.Printf("Connecting to InfluxDB server [%s]...\n", config.Url)
	influx, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     config.Url,
		Username: config.Username,
		Password: config.Password,
	})
	if err != nil {
		return fmt.Errorf("error creating InfluxDB client: %s", err.Error())
	}
	defer influx.Close()

	batchPoints, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  config.Database,
		Precision: "ns",
	})
	points, err := modemInformation.ToInfluxPoints()
	if err != nil {
		return err
	}
	batchPoints.AddPoints(points)

	fmt.Printf("Writing [%d] data points to InfluxDB database [%s]...\n", len(points), config.Database)
	err = influx.Write(batchPoints)
	if err != nil {
		return fmt.Errorf("error writing data to InfluxDB: %s", err.Error())
	}

	elapsed := time.Since(start)
	fmt.Printf("Finished writing to InfluxDB. (Took %s.)\n", elapsed)

	return nil
}

func makeAddr(hostname string, port string) string {
	// TODO: allow specifying useSsl in config
	return fmt.Sprintf("http://%s:%s", hostname, port)
}
