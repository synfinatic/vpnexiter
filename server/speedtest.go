package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os/exec"
)

type SpeedtestResults struct {
	SpeedtestUrl string // yes, this need to be here ???
	// GlobalState values
	Vendor        string
	Exit_node     string
	Exit_path     []string
	Connected_str string
	// Speedtest values
	Type               string
	Timestamp          string
	Ping_jitter        float64
	Ping_latency       float64
	Download_bandwidth float64
	Download_bytes     float64
	Download_elapsed   float64
	Upload_bandwidth   float64
	Upload_bytes       float64
	Upload_elapsed     float64
	Packetloss         float64
	Isp                string
	External_ip        string
	Server_id          float64
	Server_name        string
	Server_location    string
	Server_country     string
	Server_host        string
	Server_port        float64
	Server_ip          string
	Result_id          string
	Result_url         string
}

type SpeedtestRemote struct {
	SpeedtestUrl string
}

func run_speedtest(c echo.Context) (SpeedtestResults, error) {
	v := viper.GetViper()
	args := []string{"-f", "json"}
	if v.IsSet("serverid") {
		args = append(args, "-s", string(v.GetInt("serverid")))
	} else if v.IsSet("host") {
		args = append(args, "-o", string(v.GetInt("host")))
	}

	name := v.GetString("speedtest_cli")
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("error running %s -f json: %s", name, err)
		log.Printf("-- stderr:\n%s", stderr.String())
		errx := fmt.Errorf("Error running %s -f json: %s<p><pre>%s</pre>",
			name, err.Error(), stderr.String())
		return SpeedtestResults{}, errx
	}
	log.Printf("success running %s -f json:", name)
	log.Printf("-- stdout:\n%s", stdout.String())

	jdata := make(map[string](interface{}))
	err = json.Unmarshal(stdout.Bytes(), &jdata)
	if err != nil {
		errx := fmt.Errorf("Error parsing json: %s", err.Error())
		return SpeedtestResults{}, errx
	}

	// Instead of this mess, maybe use: https://github.com/tidwall/gjson
	ping := jdata["ping"].(map[string]interface{})
	download := jdata["download"].(map[string]interface{})
	upload := jdata["upload"].(map[string]interface{})
	iface := jdata["interface"].(map[string]interface{})
	server := jdata["server"].(map[string]interface{})
	result := jdata["result"].(map[string]interface{})

	if jdata["packetloss"] == nil {
		jdata["packetloss"] = 0.0
	}

	SR := SpeedtestResults{
		SpeedtestUrl:       "",
		Vendor:             GS.Vendor,
		Exit_node:          GS.Exit_node,
		Exit_path:          GS.Exit_path,
		Connected_str:      GS.Connected_str,
		Type:               jdata["type"].(string),
		Timestamp:          jdata["timestamp"].(string),
		Ping_jitter:        ping["jitter"].(float64),
		Ping_latency:       ping["latency"].(float64),
		Download_bandwidth: download["bandwidth"].(float64),
		Download_bytes:     download["bytes"].(float64),
		Download_elapsed:   download["elapsed"].(float64),
		Upload_bandwidth:   upload["bandwidth"].(float64),
		Upload_bytes:       upload["bytes"].(float64),
		Upload_elapsed:     upload["elapsed"].(float64),
		Packetloss:         jdata["packetloss"].(float64),
		Isp:                jdata["isp"].(string),
		External_ip:        iface["externalIp"].(string),
		Server_id:          server["id"].(float64),
		Server_name:        server["name"].(string),
		Server_location:    server["location"].(string),
		Server_country:     server["country"].(string),
		Server_host:        server["host"].(string),
		Server_port:        server["port"].(float64),
		Server_ip:          server["ip"].(string),
		Result_id:          result["id"].(string),
		Result_url:         result["url"].(string),
	}

	return SR, nil
}

func Speedtest(c echo.Context) error {
	v := viper.GetViper()
	var err error
	var SR SpeedtestResults
	if !v.IsSet("speedtest_url") {
		if !v.IsSet("speedtest_cli") {
			return c.Render(http.StatusOK, "error.html", "speedtest is not configured")
		}
		SR, err = run_speedtest(c)
		if err != nil {
			return c.Render(http.StatusOK, "error.html", err.Error())
		}
		return c.Render(http.StatusOK, "speedtest.html", SR)
	} else {
		url := v.GetString("speedtest_url")
		SR := SpeedtestRemote{SpeedtestUrl: url}
		return c.Render(http.StatusOK, "speedtest.html", SR)
	}
}
