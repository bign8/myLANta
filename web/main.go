package web

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"sync"

	"github.com/bign8/myLANta/mylanta"
)

// New constructs a new web portan handler.
func New(net *mylanta.Network) http.Handler {
	p := &Portal{
		mux: http.NewServeMux(),
		web: http.FileServer(http.Dir("web")),
		net: net,
		mem: map[string][]byte{},
		loc: sync.RWMutex{},
		tpl: template.Must(template.ParseFiles("web/index.html")),
	}
	p.mux.HandleFunc("/", p.root)
	p.mux.HandleFunc("/add", p.add)
	p.mux.HandleFunc("/get", p.get)
	p.mux.HandleFunc("/del", p.del)
	p.mux.HandleFunc("/peers", p.peers)
	return p.mux
}

// Portal is the web portal driver.
type Portal struct {
	mux *http.ServeMux
	web http.Handler
	net *mylanta.Network
	mem map[string][]byte
	loc sync.RWMutex
	tpl *template.Template
}

func showErr(w http.ResponseWriter, msg string, err error) {
	msg = msg + ": " + err.Error()
	log.Println(msg)
	http.Error(w, msg, http.StatusInternalServerError)
}

func (p *Portal) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		p.tpl = template.Must(template.ParseFiles("web/index.html")) // TODO: remove on release
		err := p.tpl.Execute(w, struct {
			Peers []mylanta.Client
			Files []string
		}{
			Peers: p.net.ActiveClients(),
			Files: p.list(),
		})
		if err != nil {
			panic(err)
		}
		return
	}
	p.web.ServeHTTP(w, r)
}

func (p *Portal) add(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("file")
	if err != nil {
		showErr(w, "problem getting file in /add", err)
		return
	}
	defer file.Close()

	// Read the full file from the browser.
	bits, err := ioutil.ReadAll(file)
	if err != nil {
		showErr(w, "problem reading file in /add", err)
		return
	}

	// Store the data in memory of the server.
	p.loc.Lock()
	p.mem[handler.Filename] = bits
	p.loc.Unlock()
	log.Printf("Loaded %q.\n", handler.Filename)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (p *Portal) get(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "required 'file' parameter missing", http.StatusExpectationFailed)
		return
	}
	data, ok := p.mem[file]
	if !ok {
		http.Error(w, "file does not exist locally", http.StatusGone)
		return
	}
	w.Write(data)
}

func (p *Portal) del(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "TODO", http.StatusNotImplemented)
}

// short term hack
func (p *Portal) list() []string {
	p.loc.RLock()
	names := make([]string, 0, len(p.mem))
	for key := range p.mem {
		names = append(names, key)
	}
	p.loc.RUnlock()
	sort.Strings(names)
	return names
}

func (p *Portal) peers(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	enc.Encode(p.net.ActiveClients())
}
