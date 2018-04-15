package web

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/bign8/myLANta/net"
)

// New constructs a new web portan handler.
func New(ctx context.Context, n *net.Network) http.Handler {
	p := &Portal{
		mux: http.NewServeMux(),
		web: http.FileServer(http.Dir("dist")),
		net: n,
		mem: map[string][]byte{},
		all: &net.FileList{
			Files: make(map[string]string),
		},
		peers: map[string]peer{},
		loc:   sync.RWMutex{},
		tpl:   template.Must(template.ParseFiles("web/index.gohtml")),
	}
	p.mux.HandleFunc("/", p.root)
	p.mux.HandleFunc("/add", p.add)
	p.mux.HandleFunc("/get", p.get)
	p.mux.HandleFunc("/del", p.del)
	p.mux.HandleFunc("/msg", p.msg)

	go networkController(ctx, n, p)
	return p.mux
}

// Portal is the web portal driver.
type Portal struct {
	mux   *http.ServeMux
	web   http.Handler
	net   *net.Network
	mem   map[string][]byte
	all   *net.FileList
	peers map[string]peer
	loc   sync.RWMutex
	tpl   *template.Template
}

func showErr(w http.ResponseWriter, msg string, err error) {
	msg = msg + ": " + err.Error()
	log.Println(msg)
	http.Error(w, msg, http.StatusInternalServerError)
}

func (p *Portal) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		p.tpl = template.Must(template.ParseFiles("web/index.gohtml")) // TODO: remove on release
		err := p.tpl.Execute(w, struct {
			Peers []string
			Files []string
		}{
			Peers: p.peerslist(),
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
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
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
	p.all.Files[handler.Filename] = "todo"
	p.net.Send(net.EncodeFileList(p.all))
	p.loc.Unlock()

	// Update client
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

// another short term hack
func (p *Portal) peerslist() []string {
	p.loc.RLock()
	names := make([]string, 0, len(p.peers))
	for _, p := range p.peers {
		if p.Alive {
			names = append(names, p.Addr.String())
		}
	}
	p.loc.RUnlock()
	sort.Strings(names)
	return names
}

func (p *Portal) msg(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")

	// Long pull http response of remaining content
	if f, ok := w.(http.Flusher); ok {
		ticker := time.NewTicker(time.Second)
		var err error
		for i := 0; err == nil; i++ {
			if _, err = fmt.Fprintf(w, `<span>%d</span><br/>`+"\n", i); err == nil { // style="display:none"
				f.Flush()
				<-ticker.C
			}
		}
		ticker.Stop()
		log.Printf("client socket closed")
	}
}
