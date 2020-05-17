package mqtt

import (
	"fmt"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/pdunnavant/modem-scraper/config"
	"github.com/pdunnavant/modem-scraper/scrape"
	"go.uber.org/zap"
)

// Publish publishes the jsonified modemInformation to
// the MQTT server configuration within the given
// configuration.
func Publish(logger *zap.Logger, config config.MQTT, modemInformation scrape.ModemInformation) error {
	start := time.Now()

	broker := makeBroker(config.Hostname, config.Port)

	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)

	client := MQTT.NewClient(opts)
	defer client.Disconnect(250)

	logger.Debug(fmt.Sprintf("connecting to MQTT server %s", broker),
		zap.String("op", "mqtt.Publish"),
	)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	logger.Debug(fmt.Sprintf("publishing to topic %s", config.Topic),
		zap.String("op", "mqtt.Publish"),
	)

	payload, err := modemInformation.ToJSON()
	if err != nil {
		return err
	}

	token := client.Publish(config.Topic, byte(0), false, payload)
	token.Wait()

	elapsed := time.Since(start)
	logger.Debug(fmt.Sprintf("finished publishing to MQTT, took %s", elapsed),
		zap.String("op", "mqtt.Publish"),
	)

	return nil
}

func makeBroker(hostname string, port string) string {
	return fmt.Sprintf("tcp://%s:%s", hostname, port)
}
