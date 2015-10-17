/*
SaaS means "Sosach Is A Service". Project focused on stream fresh WEBM from 2ch.hk/b
*/
package main

import (
	"SaaS/HTTPPlayer"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
)

// Type represents structure of configuration file(JSON)
type configFile struct {
	Port             string
	JSONUrl          string
	DownloadURL      string
	BrowserUserAgent string
	Cookie           string
	SaveDirectory    string
}

// Function read configuration and set return configFile type
func readConfig(confFilePath *string) (*configFile, error) {
	var config *configFile
	confFile, err := ioutil.ReadFile(*confFilePath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(confFile, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func main() {
	configFilePath := flag.String("conf", "config.json", "indicate path to config.json")
	flag.Parse()

	config, err := readConfig(configFilePath)
	if err != nil {
		log.Fatalln("Error on reading config file: ", err)
	}

	player, err := HTTPPlayer.NewHTTPPlayer(config.SaveDirectory, config.Cookie, config.BrowserUserAgent, config.DownloadURL, config.JSONUrl, config.Port)
	if err != nil {
		log.Fatalln("")
	}

	player.ListenAndServe()

}
