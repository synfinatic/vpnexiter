package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
)

var Konf = koanf.New(".")

/*
 * Returns a slice of Koanf config files
 */
func configFiles() []string {
	return []string{
		"./config.yaml",
		"/etc/vpnexiter/config.yaml",
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func LoadConfig(cfile string) {
	// Set Defaults
	Konf.Load(confmap.Provider(map[string]interface{}{
		"listen.address": "0.0.0.0",
		"listen.http":    8000,
		"listen.https":   -1,
		"router.mode":    "ssh",
		"router.host":    "192.168.1.1",
		"router.port":    22,
		"router.user":    "admin",
	}, "."), nil)

	if len(cfile) > 0 {
		log.Printf("Loading config file: %s", cfile)
		f := file.Provider(cfile)
		if err := Konf.Load(f, yaml.Parser()); err != nil {
			log.Fatalf("error loading config: %s", err.Error())
		}
	} else {
		for _, fname := range configFiles() {
			if fileExists(fname) {
				log.Printf("Loading config file: %s", fname)
				f := file.Provider(fname)
				if err := Konf.Load(f, yaml.Parser()); err != nil {
					log.Fatalf("error loading config: %s", err.Error())
				}
			}
		}
	}
}

type Configurations struct {
	listen    ListenConfigurations
	speedtest string
	router    RouterConfigurations
	vendors   []string
}

type ListenConfigurations struct {
	http     int
	https    int
	username string
	password string
}

type RouterConfigurations struct {
	mode           string
	config_file    string
	start_command  string
	stop_command   string
	status_command string
	// SSH Only
	host     string
	port     int
	user     string
	password string
}

func Levels(vendor string) []string {
	var empty []string
	levels := fmt.Sprintf("%s.levels", vendor)
	if Konf.Exists(levels) {
		return Konf.Strings(levels)
	} else {
		return empty
	}
}

func GetServers(vendor string, path []string) ([]string, error) {
	fullpath, err := GetPath(vendor, path)
	if err != nil {
		log.Printf("GetServers: %s / %s", vendor, fullpath)
		return nil, err
	} else {
		return Konf.Strings(fullpath), nil
	}
}

func GetPath(vendor string, path []string) (string, error) {
	fullpath := fmt.Sprintf("%s.servers", vendor)
	if len(path) > 0 {
		vars := strings.Join(path, ".")
		fullpath = fmt.Sprintf("%s.%s", fullpath, vars)
	}
	if !Konf.Exists(fullpath) {
		return "", fmt.Errorf("Invalid path: %s", fullpath)
	}
	return fullpath, nil
}

func GetPathKeys(vendor string, path []string) ([]string, error) {
	fullpath, err := GetPath(vendor, path)
	if err != nil {
		log.Printf("GetPathKeys: %s / %s", vendor, fullpath)
		return nil, err
	}
	pdata := Konf.StringMap(fullpath)
	log.Printf("fullpath %s", fullpath)
	keys := make([]string, 0, len(pdata))
	for k, _ := range pdata {
		keys = append(keys, k)
	}
	return keys, nil
}
