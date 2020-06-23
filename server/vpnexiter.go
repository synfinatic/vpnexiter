package main

import (
	"crypto/subtle"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/synfinatic/vpnexiter/vpn"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/grignaak/tribool.v1"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type GlobalState struct {
	Connected     tribool.Tribool
	Connected_str string
	Vendor        string
	Exit          string
	ExitPath      []string
	StatusOutput  string
	VPN           *vpn.VpnServer
	Vendors       map[string]*VendorConfig
}

var GS = GlobalState{
	Connected:     tribool.Maybe,
	Connected_str: "Down",
	Vendor:        "Unknown",
	Exit:          "Unselected",
	ExitPath:      []string{},
	StatusOutput:  "",
	VPN:           nil,
	Vendors:       nil,
}

func (gs *GlobalState) SetState(state tribool.Tribool) {
	log.Printf("reporting the VPN connection status is: %s\n", state)
	if state == tribool.True {
		gs.Connected = tribool.True
		gs.Connected_str = "Up"
	} else if state == tribool.False {
		gs.Connected = tribool.False
		gs.Connected_str = "Down"
	} else {
		gs.Connected = tribool.Maybe
		gs.Connected_str = "Unknown State"
	}
}

/*
 * Webpage rendering is done via html/template
 */
type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func Index(c echo.Context) error {
	type SpeedTestTypes struct {
		HasLocalSpeedtest   bool
		HasEmbededSpeedtest bool
	}
	stt := SpeedTestTypes{
		HasLocalSpeedtest:   len(Konf.String("speedtest_cli")) > 0,
		HasEmbededSpeedtest: len(Konf.String("speedtest_url")) > 0,
	}
	return c.Render(http.StatusOK, "index.html", stt)
}

func Version(c echo.Context) error {
	type Version struct {
		Version string
	}
	v := Version{"0.0.1"}
	return c.Render(http.StatusOK, "version.html", v)
}

func Status(c echo.Context) error {
	forced := c.QueryParam("forced") // forced=1
	if GS.Connected == tribool.Maybe || len(forced) > 0 {
		log.Printf("Checking status of VPN\n")
		trib, err := GS.VPN.IsUp()
		GS.SetState(trib)
		if err != nil {
			log.Printf("Error getting IsUp()")
			return c.Render(http.StatusOK, "error.html", err.Error())
		}
		buf, err := GS.VPN.Status()
		if err != nil {
			log.Printf("Error getting Status()")
			return c.Render(http.StatusOK, "error.html", err.Error())
		}
		GS.StatusOutput = buf.String()
	}
	return c.Render(http.StatusOK, "status.html", GS)
}

func SelectExit(c echo.Context) error {
	exit := c.Param("exit")
	vendor := c.Param("vendor")
	if exit == "" {
		return c.Render(http.StatusOK, "select_exit.html", GS.Vendors)
	} else {
		err := GS.VPN.UpdateConfig(vendor, exit)
		GS.Vendor = vendor
		GS.Exit = exit
		GS.SetState(tribool.False)
		if err != nil {
			return c.Render(http.StatusOK, "error.html", err.Error())
		}

		success, err := GS.VPN.Restart()
		if err != nil {
			return c.Render(http.StatusOK, "error.html", err.Error())
		}

		buf, err := GS.VPN.Status()
		if err != nil {
			log.Printf("Error getting Status()")
			return c.Render(http.StatusOK, "error.html", err.Error())
		}
		GS.StatusOutput = buf.String()

		log.Printf("Looking for exit '%s' in %v\n", exit, GS.Vendors[vendor].Servers)
		path, err := FindServerMapEntry(&GS.Vendors[vendor].Servers, exit)
		if err != nil {
			return c.Render(http.StatusOK, "error.html", err.Error())
		}
		GS.ExitPath = []string{vendor}
		GS.ExitPath = append(GS.ExitPath, path...)

		if success {
			log.Printf("VPN restart was successful\n")
			GS.SetState(tribool.True)
			return c.Redirect(http.StatusTemporaryRedirect, "/#status")
		} else {
			log.Printf("VPN restart failed\n")
			// return c.Render(http.StatusOK, "error.html", "VPN restart failed")
			return c.Redirect(http.StatusTemporaryRedirect, "/#status")
		}
	}
}

func BasicAuthHandler(username string, password string, c echo.Context) (bool, error) {
	conf_user := Konf.String("listen.username")
	conf_pass := Konf.String("listen.password")
	if subtle.ConstantTimeCompare([]byte(username), []byte(conf_user)) == 1 &&
		bcrypt.CompareHashAndPassword([]byte(conf_pass), []byte(password)) == nil {
		return true, nil
	}
	return false, nil
}

func empty_string(str string) bool {
	if str == "" {
		return true
	}
	return false
}

// speedtest -f json returns bytes/sec
func bps_to_mbps(bps float64) string {
	return fmt.Sprintf("%.02f", (bps * 8 / (1000 * 1000)))
}

func float64_to_str(val float64) string {
	return fmt.Sprintf("%.02f", val)
}

func float64_to_int(val float64) int {
	return (int)(val)
}

/*
 * Call in a goroutine because this blocks in a long sleep() loop
 * Allows us to asyncly load our GS.Vendors at startup and then
 * refresh our cache every X minutes
 */
func load_vendors() {
	GS.Vendors = LoadVendors()
	if Konf.Exists("dns_refresh_minutes") {
		if Konf.Int64("dns_refresh_minutes") >= 5 {
			for true {
				min := fmt.Sprintf("%dm", Konf.Int64("dns_refresh_minutes"))
				d, _ := time.ParseDuration(min)
				time.Sleep(d)
				GS.Vendors = LoadVendors() // do this at startup because it is slow
			}
		} else {
			log.Printf("Warning: `dns_refresh_minutes` is set, but < 5 so ignoring")
		}
	}
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger()) // debug logging: https://echo.labstack.com/middleware/logger
	LoadConfig()
	go load_vendors()

	// Enable basic auth?
	if Konf.Exists("listen.username") && Konf.Exists("listen.password") {
		e.Use(middleware.BasicAuth(BasicAuthHandler))
	}

	GS.VPN = vpn.NewVpn(Konf)

	// serve static content
	e.Static("/static", "static")
	e.File("/", "static/index.html")

	// serve templates
	funcMap := template.FuncMap{
		"StringsJoin":  strings.Join,
		"EmptyString":  empty_string,
		"BpsToMbps":    bps_to_mbps,
		"Float64ToInt": float64_to_int,
		"Float64ToStr": float64_to_str,
		// "GenerateMenu": GenerateMenu,
	}
	t := &Template{
		templates: template.Must(template.New("main").Funcs(funcMap).ParseGlob("templates/*.html")),
	}
	e.Renderer = t
	e.GET("/", Index)
	e.GET("/version", Version)
	e.GET("/status", Status)
	e.GET("/select_exit", SelectExit)
	e.GET("/select_exit/:vendor/:exit", SelectExit)

	// Lots of speed test stuff
	e.GET("/speedtest/:mode", Speedtest)
	e.GET("/speedtest/", speedtest)
	e.GET("/speedtest/host/:host", speedtest)
	e.GET("/speedtest/id/:serverid", speedtest)
	e.GET("/speedtest/servers", speedtest_servers)

	/*
	 * AJAX Calls
	 */
	// return list of vendors
	e.GET("/vendors", vendors)

	// return a map of all the exits for a vendor
	e.GET("/exits/:vendor", exits)

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
	listen := fmt.Sprintf("%s:%d", Konf.String("listen.address"), Konf.Int("listen.http"))
	e.Logger.Fatal(e.Start(listen))
}
