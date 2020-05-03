package vpnexiter

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"strings"
)

type Configurations struct {
	listen  ListenConfigurations
	mode    string
	router  RouterConfigurations
	vendors []string
}

type ListenConfigurations struct {
	http  int
	https int
}

type RouterConfigurations struct {
	ip       string
	user     string
	password string
}

func GetVendor(vendor string) (map[string]interface{}, error) {
	v := viper.GetViper()
	key := fmt.Sprintf("vendors.%s", vendor)
	if v.IsSet(key) {
		return v.GetStringMap(key), nil
	} else {
		return nil, fmt.Errorf("%s vendor is not defined", vendor)
	}
}

func Levels(vendor string) ([]string, error) {
	v := viper.GetViper()
	levels := fmt.Sprintf("%s.levels", vendor)
	if v.IsSet(levels) {
		return v.GetStringSlice(levels), nil
	} else {
		log.Printf("Levels: %s / %s", vendor, levels)
		return nil, fmt.Errorf("missing `levels` in vendor: %s", vendor)
	}
}

func Servers(vendor string) (map[string]interface{}, error) {
	v := viper.GetViper()
	path := fmt.Sprintf("%s.servers", vendor)
	if v.IsSet(path) {
		return v.GetStringMap(path), nil
	} else {
		log.Printf("Servers: %s", vendor)
		return nil, fmt.Errorf("invalid vendor: %s", vendor)
	}
}

func GetServers(vendor string, path []string) ([]string, error) {
	v := viper.GetViper()
	fullpath, err := GetPath(vendor, path)
	if err != nil {
		log.Printf("GetServers: %s / %s", vendor, fullpath)
		return nil, err
	} else {
		return v.GetStringSlice(fullpath), nil
	}
}

func FindLevel(vendor, string, level string) (int, error) {
	l, err := Levels(vendor)
	if err != nil {
		return -1, err
	}
	for i, n := range l {
		if n == level {
			return i, nil
		}
	}
	log.Printf("FindLevel: %s / %s", vendor, level)
	return -1, fmt.Errorf("Level %s doesn't exist in %s", level, vendor)
}

func GetPath(vendor string, path []string) (string, error) {
	v := viper.GetViper()
	vars := strings.Join(path, ".")
	fullpath := fmt.Sprintf("%s.servers.%s", vendor, vars)
	if !v.IsSet(fullpath) {
		return "", fmt.Errorf("Invalid path: %s", fullpath)
	}
	return fullpath, nil
}

func GetPathKeys(vendor string, path []string) ([]string, error) {
	v := viper.GetViper()
	fullpath, err := GetPath(vendor, path)
	if err != nil {
		log.Printf("GetPathValues: %s / %s", vendor, fullpath)
		return nil, err
	} else {
		return v.GetStringSlice(fullpath), nil
	}
}
