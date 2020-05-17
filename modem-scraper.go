package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/pdunnavant/modem-scraper/boltdb"
	"github.com/pdunnavant/modem-scraper/config"
	"github.com/pdunnavant/modem-scraper/influxdb"
	"github.com/pdunnavant/modem-scraper/mqtt"
	"github.com/pdunnavant/modem-scraper/scrape"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// BuildVersion is the version of the binary, and is set with ldflags at build time.
var BuildVersion = "UNKNOWN"

// CliInputs holds the data passed in via CLI parameters
type CliInputs struct {
	BuildVersion string
	Config       string
	ShowVersion  bool
}

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Println("{\"op\": \"main\", \"level\": \"fatal\", \"msg\": \"failed to initiate logger\"}")
		panic(err)
	}
	defer logger.Sync()

	cliInputs := CliInputs{
		BuildVersion: BuildVersion,
	}
	flags := flag.NewFlagSet("modem-scraper", 0)
	flags.StringVar(&cliInputs.Config, "config", "config.yaml", "Set the location for the YAML config file")
	flags.BoolVar(&cliInputs.ShowVersion, "version", false, "Print the version of modem-script")
	flags.Parse(os.Args[1:])

	if cliInputs.ShowVersion {
		fmt.Println(cliInputs.BuildVersion)
		os.Exit(0)
	}

	configuration, err := parseConfiguration(cliInputs.Config)
	if err != nil {
		logger.Fatal("failed to parse configuration",
			zap.String("op", "main"),
			zap.Error(err),
		)
		panic(err)
	}

	c := cron.New()
	c.AddFunc(configuration.Polling.Schedule, func() {
		logger.Debug("waking up",
			zap.String("op", "main"),
		)
		modemInformation, err := scrape.Scrape(logger, *configuration)
		if err != nil {
			logger.Error("failed to scrape modem information",
				zap.String("op", "main"),
				zap.Error(err),
			)
			return
		}

		modemInformation, err = boltdb.PruneEventLogs(configuration.BoltDB, *modemInformation)
		if err != nil {
			logger.Error("failed to prune event logs from BoltDB",
				zap.String("op", "main"),
				zap.Error(err),
			)
			return
		}

		if configuration.InfluxDB.Enabled {
			err = influxdb.Publish(logger, configuration.InfluxDB, *modemInformation)
			if err != nil {
				logger.Error("failed to write data to InfluxDB",
					zap.String("op", "main"),
					zap.Error(err),
				)
				return
			}
		}

		if configuration.MQTT.Enabled {
			err = mqtt.Publish(logger, configuration.MQTT, *modemInformation)
			if err != nil {
				logger.Error("failed to write data to MQTT",
					zap.String("op", "main"),
					zap.Error(err),
				)
				return
			}
		}

		err = boltdb.UpdateEventLogs(logger, configuration.BoltDB, *modemInformation)
		if err != nil {
			logger.Error("failed to update event logs in BoltDB",
				zap.String("op", "main"),
				zap.Error(err),
			)
			return
		}

		logger.Debug("going back to sleep",
			zap.String("op", "main"),
		)
	})
	go c.Start()

	// Wait forever, but just for an OS interrupt/kill.
	logger.Debug("started",
		zap.String("op", "main"),
	)
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)
	<-sig
}

func parseConfiguration(configPath string) (*config.Configuration, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Error reading config file, %s", err)
	}

	var configuration config.Configuration
	err := viper.Unmarshal(&configuration)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %s", err)
	}

	return &configuration, nil
}
