package influxdb

import (
	"fmt"
	"time"

	_ "github.com/influxdata/influxdb1-client" // this is important because of a bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/pdunnavant/modem-scraper/config"
	"github.com/pdunnavant/modem-scraper/scrape"
	"go.uber.org/zap"
)

// Publish publishes the data within modemInformation to
// the InfluxDB server configuration within the given
// configuration.
func Publish(logger *zap.Logger, config config.InfluxDB, modemInformation scrape.ModemInformation) error {
	start := time.Now()

	logger.Debug(fmt.Sprintf("connecting to InfluxDB server %s", config.Url),
		zap.String("op", "influxdb.Publish"),
	)

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

	logger.Debug(fmt.Sprintf("writing %d data points to InfluxDB database %s", len(points), config.Database),
		zap.String("op", "influxdb.Publish"),
	)
	err = influx.Write(batchPoints)
	if err != nil {
		return fmt.Errorf("error writing data to InfluxDB: %s", err.Error())
	}

	elapsed := time.Since(start)
	logger.Debug(fmt.Sprintf("finished writing to InfluxDB, took %s", elapsed),
		zap.String("op", "influxdb.Publish"),
	)

	return nil
}
