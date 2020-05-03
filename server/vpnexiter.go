package main

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"github.com/synfinatic/vpnexiter/vpnexiter"
	"log"
	"net/http"
)

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
	viper.SetDefault("router.ip", "192.168.1.1")
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
	e.GET("/", func(c echo.Context) error {
		vendor := c.QueryParam("vendor")
		levels, err := vpnexiter.Levels(vendor)
		if err != nil {
			return c.String(http.StatusNotFound, fmt.Sprintf("Invalid vendor: %s", vendor))
		}

		log.Printf("%s has %d levels", vendor, len(levels))
		for i, l := range levels {
			log.Printf("Level %d = %s", i, l)
		}
		l0 := c.QueryParam("l0")
		l1 := c.QueryParam("l1")
		path := []string{l0, l1}
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
		jdata, err := json.Marshal(slist)
		if err != nil {
			return c.String(http.StatusNotFound, err.Error())
		}

		return c.String(http.StatusOK, string(jdata))
		//		return c.String(http.StatusOK, servers[0])
	})
	e.Logger.Fatal(e.Start(":5000"))
}
