package config

// Configuration holds all configuration for modem-scraper.
type Configuration struct {
	IP           string
	PollSchedule string
	MQTT         MQTT
	InfluxDB     InfluxDB
	BoltDB       BoltDB
}

// MQTT holds MQTT connection configuration.
type MQTT struct {
	Enabled  bool
	Hostname string
	Port     string
	Username string
	Password string
	Topic    string
	ClientID string
}

// InfluxDB holds InfluxDB connection configuration.
type InfluxDB struct {
	Enabled  bool
	Hostname string
	Port     string
	Database string
	Username string
	Password string
}

// BoltDB holds BoltDB configuration.
type BoltDB struct {
	Path string
}
