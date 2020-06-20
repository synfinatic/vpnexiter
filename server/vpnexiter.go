package main

import (
	"crypto/subtle"
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"io"
	"net/http"
	"strings"
)

type GlobalState struct {
	Connected     bool
	Connected_str string
	Vendor        string
	Exit_node     string
	Exit_path     []string
}

var GS = GlobalState{
	Connected:     false,
	Connected_str: "Down",
	Vendor:        "",
	Exit_node:     "",
	Exit_path:     []string{},
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
	return c.Render(http.StatusOK, "status.html", GS)
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

func main() {
	e := echo.New()
	e.Use(middleware.Logger()) // debug logging: https://echo.labstack.com/middleware/logger
	LoadConfig()

	// Enable basic auth?
	if Konf.Exists("listen.username") && Konf.Exists("listen.password") {
		e.Use(middleware.BasicAuth(BasicAuthHandler))
	}

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
	e.GET("/speedtest/:mode", Speedtest)

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
	e.GET("/speedtest/", speedtest)
	e.GET("/speedtest/host/:host", speedtest)
	e.GET("/speedtest/id/:serverid", speedtest)
	e.GET("/speedtest/servers", speedtest_servers)
	e.GET("/update/:vendor/:ipaddr", update)
	e.Logger.Fatal(e.Start(":5000"))
}
