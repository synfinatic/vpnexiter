package main

/*
 * These are all the methods used as AJAX calls by the
 * main webpages.
 */

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

/*
 * Return a list of VPN Vendors
 */
func vendors(c echo.Context) error {
	venlist := Konf.Strings("vendors")
	return c.JSONPretty(http.StatusOK, venlist, " ")
}

/*
 * For tie given vendor, return a list of Levels
 */
func levels(c echo.Context) error {
	vendor := c.Param("vendor")
	l := Levels(vendor)
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
	keys, err := GetPathKeys(vendor, path)
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
	e := exec.Command(Konf.String("speedtest"), "--servers", "-f", "json-pretty")
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
	// ipaddr := c.Param("ipaddr")
	serverid := c.Param("serverid")

	args := []string{"-f", "json-pretty"}
	if len(serverid) != 0 {
		log.Printf("Using speedtest server: %s", serverid)
		args = append(args, "--server-id", serverid)
	} else {
		log.Printf("Letting speedtest.net pick our server...")
	}
	e := exec.Command(Konf.String("speedtest"), args...)

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
	err := Update(vendor, ipaddr)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSONPretty(http.StatusOK, "OK", " ")
}

/*
 * Returns the server(s) for the given vendor and level
 */
func servers(c echo.Context) error {
	vendor := c.Param("vendor")
	levels := Levels(vendor)
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

	servers, err := GetServers(vendor, path)
	if err != nil {
		return c.String(http.StatusNotFound, err.Error())
	}
	if len(servers) == 0 {
		return c.String(http.StatusNotFound, fmt.Sprintf("%s has no servers", vendor))
	}
	slist, err := Server2ServerList(vendor, path)
	if err != nil {
		return c.String(http.StatusNotFound, err.Error())
	}

	return c.JSONPretty(http.StatusOK, slist, " ")
}

func exits(c echo.Context) error {
	exit_map, _, err := _exits(c)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	jdata, err := json.Marshal(exit_map)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSONPretty(http.StatusOK, jdata, " ")
}

func _exits(c echo.Context) (interface{}, int, error) {
	vendor := c.Param("vendor")
	levels := Levels(vendor)
	vpath := vendor + ".servers"
	lvls := len(levels)
	switch lvls {
	case 0:
		log.Printf("getting servers for %s", vendor)
		data := Konf.Strings(vpath)
		return data, 0, nil
	case 1, 2, 3, 4, 5:
		data := walk_levels(vpath, lvls, 1)
		return data, lvls, nil
	default:
		return nil, 0, fmt.Errorf("Invalid number of vendor levels for %s", vendor)
	}
}

/*
 * Recursively walk the vendor.servers hash map and return
 */

func walk_levels(path string, levels int, depth int) interface{} {
	if levels <= depth {
		// this level is a map[string]interface
		us := make(map[string]interface{}, 0)
		keys := Konf.MapKeys(path)
		for _, key := range keys {
			newpath := path + "." + key
			fmt.Printf("going deeper: %s\n", newpath)
			us[key] = walk_levels(newpath, levels, depth+1)
		}
		return us
	} else {
		// this level is the final level and a map[string][]string
		us := Konf.StringsMap(path)
		return us
	}
}

func SelectExit(c echo.Context) error {
	exit := c.Param("exit")
	if exit == "" {
		vendors := LoadVendors()
		return c.Render(http.StatusOK, "select_exit.html", vendors)
	} else {
		// FIXME: actually do something useful here
		return Version(c)
	}
}
