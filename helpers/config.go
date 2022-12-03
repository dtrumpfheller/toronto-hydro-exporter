package helpers

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	InfluxDB       InfluxDB     `yaml:"influxDB"`
	TorontoHydro   TorontoHydro `yaml:"torontoHydro"`
	SleepDuration  int          `yaml:"sleepDuration"`
	LookDaysInPast int          `yaml:"lookDaysInPast"`
}

type InfluxDB struct {
	URL          string `yaml:"url"`
	Token        string `yaml:"token"`
	Organization string `yaml:"organization"`
	Bucket       string `yaml:"bucket"`
}

type TorontoHydro struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Meter    string `yaml:"meter"`
	Id       string `yaml:"id"`
	Mock     bool   `yaml:"mock"`
}

func ReadConfig(configFile string) Config {
	var appConfig Config

	// check if specific
	if len(configFile) == 0 {
		log.Fatalln("Configuration file not specified!")
	}

	// check file ending
	if filepath.Ext(configFile) != ".yml" {
		log.Fatalln("Configuration file is not YAML!")
	}

	// check if file exists
	if !fileExists(configFile) {
		log.Fatalln("Configuration file doesn't exist!")
	}

	// load file into config object
	f, err := os.Open(configFile)
	if err != nil {
		log.Fatalln("Error reading the configuration file!")
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&appConfig)

	if err != nil {
		log.Fatalln("Error reading the configuration file! Is it valid YAML?")
	}

	return appConfig
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
