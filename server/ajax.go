package main

/*
 * These are all the methods used as AJAX calls by the
 * main webpages.
 */

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/spf13/viper"
	"github.com/synfinatic/vpnexiter/vpnexiter"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

/*
 * Return a list of VPN Vendors
 */
func vendors(c echo.Context) error {
	v := viper.GetViper()
	venlist := v.GetStringSlice("vendors")
	return c.JSONPretty(http.StatusOK, venlist, " ")
}

/*
 * For tie given vendor, return a list of Levels
 */
func levels(c echo.Context) error {
	vendor := c.Param("vendor")
	l := vpnexiter.Levels(vendor)
	return c.JSONPretty(http.StatusOK, l, " ")
}

/*
func url(command string, c echo.Context) string {
	var url_parts []string
	for _, part := range c.ParamNames() {
		url_parts = append(url_parts, c.Param(part))
	}
	u := "/servers/" + strings.Join(url_parts, "/")
	log.Printf("New url: %s", u)
	return u
}
*/

/*
 * For the given vendor/level, return the keys of the level below
 */
func level(c echo.Context) error {
	vendor := c.Param("vendor")
	var path []string
	log.Printf("param names: %s", string(strings.Join(c.ParamNames(), ", ")))
	for _, pname := range c.ParamNames() {
		if pname == "vendor" {
			continue
		}
		path = append(path, strings.ReplaceAll(c.Param(pname), "+", " "))
	}
	keys, err := vpnexiter.GetPathKeys(vendor, path)
	if err != nil {
		return c.String(http.StatusNotFound, err.Error())
	}
	if len(keys) == 0 {
		return servers(c)
	}
	return c.JSONPretty(http.StatusOK, keys, " ")
}

/*
 * Grab a list of "local" speedtest servers
 */
func speedtest_servers(c echo.Context) error {
	v := viper.GetViper()

	e := exec.Command(v.GetString("speedtest"), "--servers", "-f", "json-pretty")
	output, err := e.Output()
	if err != nil {
		log.Printf("error at output")
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, string(output))
}

/*
 * Run a speedtest test.
 * Must: Provide the IP address of the VPN server you're egressing from
 * Optionally: Pick the speedtest.net serverid to egress from
 */
func speedtest(c echo.Context) error {
	v := viper.GetViper()
	// ipaddr := c.Param("ipaddr")
	serverid := c.Param("serverid")

	args := []string{"-f", "json-pretty"}
	if len(serverid) != 0 {
		log.Printf("Using speedtest server: %s", serverid)
		args = append(args, "--server-id", serverid)
	} else {
		log.Printf("Letting speedtest.net pick our server...")
	}
	e := exec.Command(v.GetString("speedtest"), args...)

	output, err := e.Output()
	if err != nil {
		log.Printf("error at output")
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, string(output))
}

/*
 * Change egress to the provided vendor/VPN gateway
 */
func update(c echo.Context) error {
	vendor := c.Param("vendor")
	ipaddr := c.Param("ipaddr")
	err := vpnexiter.Update(vendor, ipaddr)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSONPretty(http.StatusOK, "OK", " ")
}

/*
 * Returns the server(s) for the given vendor and level
 */
func servers(c echo.Context) error {
	vendor := c.Param("vendor")
	levels := vpnexiter.Levels(vendor)
	log.Printf("%s has %d levels", vendor, len(levels))
	log.Printf("param names: %s", string(strings.Join(c.ParamNames(), ", ")))
	var path []string
	for i, pname := range c.ParamNames() {
		if pname == "vendor" {
			continue
		}
		log.Printf("adding %d %s to path", i, pname)
		path = append(path, strings.ReplaceAll(c.Param(pname), "+", " "))
	}

	servers, err := vpnexiter.GetServers(vendor, path)
	if err != nil {
		return c.String(http.StatusNotFound, err.Error())
	}
	if len(servers) == 0 {
		return c.String(http.StatusNotFound, fmt.Sprintf("%s has no servers", vendor))
	}
	slist, err := vpnexiter.Server2ServerList(vendor, path)
	if err != nil {
		return c.String(http.StatusNotFound, err.Error())
	}

	return c.JSONPretty(http.StatusOK, slist, " ")
}
