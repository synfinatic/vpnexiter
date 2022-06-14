package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/labstack/echo/v4"
)

type SpeedtestResults struct {
	SpeedtestURL string // yes, this need to be here ???
	// GlobalState values
	Vendor       string
	Exit         string
	ExitPath     []string
	ConnectedStr string
	// Speedtest values
	Type              string
	Timestamp         string
	PingJitter        float64
	PingLatency       float64
	DownloadBandwidth float64
	DownloadBytes     float64
	DownloadElapsed   float64
	UploadBandwidth   float64
	UploadBytes       float64
	UploadElapsed     float64
	Packetloss        float64
	Isp               string
	ExternalIP        string
	ServerID          float64
	ServerName        string
	ServerLocation    string
	ServerCountry     string
	ServerHost        string
	ServerPort        float64
	ServerIP          string
	ResultID          string
	ResultURL         string
}

type SpeedtestRemote struct {
	SpeedtestURL string
}

func runSpeedtest(c echo.Context) (SpeedtestResults, error) {
	args := []string{"-f", "json"}
	if Konf.Exists("serverid") {
		args = append(args, "-s", fmt.Sprintf("%d", Konf.Int("serverid")))
	} else if Konf.Exists("host") {
		args = append(args, "-o", fmt.Sprintf("%d", Konf.Int("host")))
	}

	name := Konf.String("speedtest_cli")
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
		SpeedtestURL:      "",
		Vendor:            GS.Vendor,
		Exit:              GS.Exit,
		ExitPath:          GS.ExitPath,
		ConnectedStr:      GS.ConnectedStr,
		Type:              jdata["type"].(string),
		Timestamp:         jdata["timestamp"].(string),
		PingJitter:        ping["jitter"].(float64),
		PingLatency:       ping["latency"].(float64),
		DownloadBandwidth: download["bandwidth"].(float64),
		DownloadBytes:     download["bytes"].(float64),
		DownloadElapsed:   download["elapsed"].(float64),
		UploadBandwidth:   upload["bandwidth"].(float64),
		UploadBytes:       upload["bytes"].(float64),
		UploadElapsed:     upload["elapsed"].(float64),
		Packetloss:        jdata["packetloss"].(float64),
		Isp:               jdata["isp"].(string),
		ExternalIP:        iface["externalIp"].(string),
		ServerID:          server["id"].(float64),
		ServerName:        server["name"].(string),
		ServerLocation:    server["location"].(string),
		ServerCountry:     server["country"].(string),
		ServerHost:        server["host"].(string),
		ServerPort:        server["port"].(float64),
		ServerIP:          server["ip"].(string),
		ResultID:          result["id"].(string),
		ResultURL:         result["url"].(string),
	}

	return SR, nil
}

func Speedtest(c echo.Context) error {
	mode := c.Param("mode")
	// If we don't have a speedtest_url set, use the speedtest_cli
	if mode == "embeded" {
		if !Konf.Exists("speedtest_url") {
			return c.Render(http.StatusOK, "error.html", "Embeded speedtest is not configured")
		}
		url := Konf.String("speedtest_url")
		SR := SpeedtestRemote{SpeedtestURL: url}
		return c.Render(http.StatusOK, "speedtest.html", SR)
	} else if mode == "server" {
		if !Konf.Exists("speedtest_cli") {
			return c.Render(http.StatusOK, "error.html", "CLI speedtest is not configured")
		}
		SR, err := runSpeedtest(c)
		if err != nil {
			return c.Render(http.StatusOK, "error.html", err.Error())
		}
		// render our (ugly as sin) page with results
		return c.Render(http.StatusOK, "speedtest.html", SR)
	}
	return nil
}
