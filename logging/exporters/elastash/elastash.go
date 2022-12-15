// Very trivial HTTP API sender, for testing telemetry upgrades
// This is for testing POST to ElasticSearch v6, v7, and Logstash
// Adapted from portions of olivere/elastic

package elastash

const (
	Version = "0.0.0"
)

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{}) 
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
}

