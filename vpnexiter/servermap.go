package vpnexiter

import (
	"bytes"
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"html/template"
	"log"
	"strings"
)

type ServerMap struct {
	List []string
	Map  map[string]ServerMap
}

func newServerMap() *ServerMap {
	return &ServerMap{
		List: []string{},
		Map:  map[string]ServerMap{},
	}
}

/*
 * By default, new ServerMap's are "undefined".  That is to say
 * they are neither a Map or a List.  Once you call addMap, addList
 * or appendList, then it becomes that type
 */
func (sm ServerMap) isList() bool {
	return len(sm.List) > 0
}

func (sm ServerMap) isMap() bool {
	return len(sm.Map) > 0
}

func (sm ServerMap) getList() ([]string, error) {
	if sm.isList() {
		return sm.List, nil
	} else if sm.isMap() {
		return nil, fmt.Errorf("ServerMap is a Map, not List")
	}
	return nil, fmt.Errorf("Can't getList() because ServerMap is undefined")
}

func (sm *ServerMap) getMap() (map[string]ServerMap, error) {
	if sm.isList() {
		return nil, fmt.Errorf("ServerMap is a List")
	} else if sm.isMap() {
		return sm.Map, nil
	}
	return nil, fmt.Errorf("Can't getMap() because ServerMap is undefined")
}

func (sm *ServerMap) addList(key string, servers []string) error {
	if sm.isList() {
		return fmt.Errorf("Can not add a key to a List")
	}

	sm.Map[key] = ServerMap{List: servers}
	return nil
}

func (sm *ServerMap) appendList(servers []string) error {
	if sm.isMap() {
		return fmt.Errorf("Can't append a List since we are a Map")
	}

	sm.List = append(sm.List, servers...)
	return nil
}

func (sm *ServerMap) addMap(key string, mdata *ServerMap) error {
	if sm.isList() {
		return fmt.Errorf("Can't add a Map to a List")
	}
	sm.Map[key] = *mdata
	return nil
}

/*
 * Walks a server map and prints it out
 */
func walkServerMap(sm ServerMap) {
	_walkServerMap(sm, 0)
}

// helper function for walkServerMap()
func _walkServerMap(sm ServerMap, depth int) {
	if sm.isList() {
		l, _ := sm.getList()
		for _, server := range l {
			fmt.Printf("%s- %s\n", strings.Repeat("\t", depth), server)
		}
	} else if sm.isMap() {
		m, _ := sm.getMap()
		for key, sm := range m {
			fmt.Printf("%s%s:\n", strings.Repeat("\t", depth), key)
			_walkServerMap(sm, depth+1)
		}
	} else {
		log.Printf("Undefined ServerMap: %v at depth %d", sm, depth)
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
		if head.isMap() {
			m, _ := head.getMap()
			x := m[key]
			insertServers(&x, loc, servers)
		} else {
			m := &ServerMap{}
			err := head.addMap(key, m)
			if err != nil {
				return err
			}
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

	if sm.isList() {
		l, _ := sm.getList()
		return l, nil
	} else if sm.isMap() {
		next := location[0]
		loc := location[1:len(location)]
		s, _ := sm.getMap()
		return s[next].getServers(loc)
	}

	return nil, fmt.Errorf("Unable to locate remainder: %s", strings.Join(location, "."))
}

/*
 * Needed for importing into the html/template
 */
func (sm ServerMap) GenHTMLTemplate() (template.HTML, error) {
	s, err := sm.GenHTML()
	if err != nil {
		return "", err
	}
	return template.HTML(s), nil
}

/*
 * Generates a HTML tree representation of a ServerMap
 */
func (sm ServerMap) GenHTML() (string, error) {
	var html bytes.Buffer

	list_tmpl, _ := template.New("list").Parse(
		heredoc.Doc(
			`{{range $name := .}}
	<li>
		<div>{{$name}}</div>
	</li>
{{end}}`,
		))

	if sm.isList() {
		l, _ := sm.getList()
		err := list_tmpl.Execute(&html, l)
		if err != nil {
			return "", err
		}
	} else if sm.isMap() {
		m, _ := sm.getMap()
		for key, value := range m {
			header := fmt.Sprintf("<li><div>%s</div><ul>", key)
			html.Write([]byte(header))
			body, err := value.GenHTML()
			if err != nil {
				return "", err
			}
			tail := fmt.Sprintf("%s</ul></li>\n", body)
			html.Write([]byte(tail))
		}
	}
	return html.String(), nil
}
