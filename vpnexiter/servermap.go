package vpnexiter

import (
	"bytes"
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"html/template"
	"log"
	"sort"
	"strings"
	"sync"
)

/*
 * This data struct represents a node in our tree of VPN servers.
 * The mutex is there incase we can ever get multithreaded DNS queries
 * to work.
 *
 * LinkKeys is used to inform GenHTML() if we want the keys of the hash
 * to be hyperlinks (normally, just the values in a list are).
 *
 * Everything else should be pretty self explainatory!
 */
type ServerMap struct {
	mux      sync.Mutex // not really needed!
	Vendor   string
	LinkKeys bool
	Parent   *ServerMap
	Name     string
	List     []string
	Map      map[string]ServerMap
}

func newServerMap(parent *ServerMap, name string, vendor string, link_keys bool) *ServerMap {
	return &ServerMap{
		mux:      sync.Mutex{},
		Parent:   parent,
		Name:     name,
		Vendor:   vendor,
		LinkKeys: link_keys,
		List:     []string{},
		Map:      map[string]ServerMap{},
	}
}

/*
 * By default, new ServerMap's are "undefined".  That is to say
 * they are neither a Map or a List.  Once you call addMap, addList
 * or appendList, then it becomes that type
 */
func (sm ServerMap) hasList() bool {
	return len(sm.List) > 0
}

func (sm ServerMap) hasMap() bool {
	return len(sm.Map) > 0
}

func (sm ServerMap) getList() []string {
	return sm.List
}

func (sm *ServerMap) getMap() map[string]ServerMap {
	return sm.Map
}

func (sm *ServerMap) addList(key string, servers []string) {
	// someday, maybe we'll even be able to use this mutex
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.Map[key] = ServerMap{List: servers}
}

func (sm *ServerMap) appendList(servers []string) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.List = append(sm.List, servers...)
}

func (sm *ServerMap) addMap(key string, mdata *ServerMap) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.Map[key] = *mdata
}

/*
 * Returns the path to our current node
 */
func (sm ServerMap) getPath() []string {
	path := []string{sm.Name}
	p := sm.Parent
	// walk up our tree
	for p != nil {
		path = append(path, p.Name)
		p = sm.Parent
	}
	// standard in-place reverse slice elements
	last := len(path) - 1
	for i := 0; i < len(path)/2; i++ {
		path[i], path[last-1] = path[last-1], path[i]
	}
	return path
}

/*
 * Walks a server map and prints it out
 */
func walkServerMap(sm ServerMap) {
	_walkServerMap(sm, 0)
}

// helper function for walkServerMap()
func _walkServerMap(sm ServerMap, depth int) {
	if sm.hasList() {
		l := sm.getList()
		for _, server := range l {
			fmt.Printf("%s- %s\n", strings.Repeat("\t", depth), server)
		}
	}
	if sm.hasMap() {
		m := sm.getMap()
		for key, sm := range m {
			fmt.Printf("%s%s:\n", strings.Repeat("\t", depth), key)
			_walkServerMap(sm, depth+1)
		}
	}
}

/*
 * Inserts a list of servers to the provided location, using the given ServerMap
 * assumes the `head` is the head of the map :)
 */
func insertServers(head *ServerMap, location []string, servers []string) error {
	// if our location includes the vendor + vendor_name, strip that
	if location[0] == "vendor" {
		location = location[2:len(location)]
	}

	if len(location) == 0 {
		// only happens if `vendor.<name>` is a list
		head.appendList(servers)
	} else if len(location) == 1 {
		// `vendor.<name>.<region>`[<city>] = servers
		head.addList(location[0], servers)
	} else {
		// recurse
		key := location[0]
		loc := location[1:len(location)]
		if head.hasMap() {
			m := head.getMap()
			x := m[key]
			insertServers(&x, loc, servers)
		} else {
			m := &ServerMap{}
			head.addMap(key, m)
			insertServers(m, loc, servers)
		}
	}
	return nil
}

/*
 * Get a list of servers using the provided location
 * if the location doesn't map to a list of servers, return error
 */
func (sm ServerMap) getServers(location []string) ([]string, error) {
	// if our location includes the vendor + vendor_name, strip that
	if location[0] == "vendor" {
		location = location[2:len(location)]
	}

	if sm.hasList() {
		l := sm.getList()
		return l, nil
	} else if sm.hasMap() {
		next := location[0]
		loc := location[1:len(location)]
		s := sm.getMap()
		return s[next].getServers(loc)
	}

	return nil, fmt.Errorf("Unable to locate remainder: %s", strings.Join(location, "."))
}

/*
 * Needed for importing into the html/template
 */
func (sm ServerMap) GenHTMLTemplate() (template.HTML, error) {
	// FIXME: is there some way to pass the url in from the select_exit.html template?
	s, err := sm.GenHTML("/select_exit", sm.Vendor)
	if err != nil {
		log.Fatal(err.Error())
		return "", err
	}
	return template.HTML(s), nil
}

func (sm ServerMap) mapKeyToLabel(key string) (string, error) {
	if !sm.LinkKeys || len(sm.Vendor) == 0 {
		return key, nil
	} else {
		return fmt.Sprintf(`<a href="select_exit/%s/%s">%s</a>`, sm.Vendor, key, key), nil
	}
}

/*
 * Generates a HTML tree representation of a ServerMap
 */
func (sm ServerMap) GenHTML(baseurl string, vendor string) (string, error) {
	var html bytes.Buffer

	/*
	 * Mix Sprintf & template because you can't access a top
	 * level variable (baseurl) from inside a loop (range exits)
	 * because html.template kinda sucks
	 */
	list_tmpl, _ := template.New("server_list").Parse(
		fmt.Sprintf(
			heredoc.Doc(
				`{{range $name := .}}
	<li>
		<div><a href="%s/%s/{{$name}}">{{$name}}</a></div>
	</li>
{{end}}`,
			),
			baseurl, vendor),
	)

	if sm.hasList() {
		l := sm.getList()
		if len(l) > 1 {
			err := list_tmpl.Execute(&html, l)
			if err != nil {
				log.Fatal(err.Error())
			}
		} else {
			x := l[0]
			buf := fmt.Sprintf(`<a href="%s/%s/%s">%s</a>`, baseurl, vendor, x, x)
			html.Write([]byte(buf))
		}
	}

	if sm.hasMap() {
		m := sm.getMap()
		mapkeys := []string{}
		for key, _ := range m {
			mapkeys = append(mapkeys, key)
		}
		sort.Strings(mapkeys)
		for _, key := range mapkeys {
			value := m[key]
			label, err := sm.mapKeyToLabel(key)
			if err != nil {
				log.Fatal(err.Error())
			}
			header := fmt.Sprintf("<li><div>%s</div><ul>", label)
			html.Write([]byte(header))
			body, err := value.GenHTML(baseurl, sm.Vendor)
			if err != nil {
				log.Fatal(err.Error())
			}
			tail := fmt.Sprintf("%s</ul></li>\n", body)
			html.Write([]byte(tail))
		}
	}
	return html.String(), nil
}
