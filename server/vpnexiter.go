package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"github.com/synfinatic/vpnexiter/vpnexiter"
	"html/template"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func Example(c echo.Context) error {
	return c.Render(http.StatusOK, "hello", "world")
}

func vendors(c echo.Context) error {
	v := viper.GetViper()
	venlist := v.GetStringSlice("vendors")
	return c.JSONPretty(http.StatusOK, venlist, " ")
}

func levels(c echo.Context) error {
	vendor := c.Param("vendor")
	l := vpnexiter.Levels(vendor)
	return c.JSONPretty(http.StatusOK, l, " ")
}

func url(command string, c echo.Context) string {
	var url_parts []string
	for _, part := range c.ParamNames() {
		url_parts = append(url_parts, c.Param(part))
	}
	u := "/servers/" + strings.Join(url_parts, "/")
	log.Printf("New url: %s", u)
	return u
}

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

func speedtest(c echo.Context) error {
	v := viper.GetViper()
	e := exec.Command(v.GetString("speedtest"), "-f", "json-pretty")
	output, err := e.Output()
	if err != nil {
		log.Printf("error at output")
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, string(output))
}

func update(c echo.Context) error {
	v := viper.GetViper()
	vendor := v.Param("vendor")
	ipaddr := v.Param("ipaddr")
	jdata, err := vpnexiter.Update(vendor, ipaddr)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSONPretty(http.StatusOK, "OK", " ")
}

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

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/vpnexiter")
	viper.AutomaticEnv()

	// Set Defaults
	viper.SetDefault("listen.http", 8000)
	viper.SetDefault("listen.https", -1)
	viper.SetDefault("mode", "ssh")
	viper.SetDefault("router.host", "192.168.1.1")
	viper.SetDefault("router.port", 22)
	viper.SetDefault("router.user", "admin")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

	var vconf vpnexiter.Configurations

	err := viper.Unmarshal(&vconf)
	if err != nil {
		fmt.Printf("Unable to decode into struct, %v", err)
	}

	e := echo.New()

	// serve static content
	e.Static("/static", "static")

	// serve templates
	t := &Template{
		templates: template.Must(template.ParseGlob("templates/*")),
	}
	e.Renderer = t
	e.GET("/example.html", Example)

	// return list of vendors
	e.GET("/vendors", vendors)

	// For the given vendor, return the levels
	e.GET("/levels/:vendor", levels)

	// For the given vendor, return the keys below the given level
	e.GET("/level/:vendor", level)
	e.GET("/level/:vendor/:l0", level)
	e.GET("/level/:vendor/:l0/:l1", level)
	e.GET("/level/:vendor/:l0/:l1/:l2", level)
	e.GET("/level/:vendor/:l0/:l1/:l2/:l3", level)
	e.GET("/level/:vendor/:l0/:l1/:l2/:l3/:l4", level)
	e.GET("/level/:vendor/:l0/:l1/:l2/:l3/:l4/:l5", level)

	// For the given vendor, return the servers for the given key
	e.GET("/servers/:vendor", servers)
	e.GET("/servers/:vendor/:l0", servers)
	e.GET("/servers/:vendor/:l0/:l1", servers)
	e.GET("/servers/:vendor/:l0/:l1/:l2", servers)
	e.GET("/servers/:vendor/:l0/:l1/:l2/:l3", servers)
	e.GET("/servers/:vendor/:l0/:l1/:l2/:l3/:l4", servers)
	e.GET("/servers/:vendor/:l0/:l1/:l2/:l3/:l4/:l5", servers)
	e.GET("/speedtest/:ipaddr", speedtest)
	e.GET("/update/:vendor/:ipaddr", update)
	e.Logger.Fatal(e.Start(":5000"))
}
