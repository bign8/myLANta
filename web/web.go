package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
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
	p.mux.HandleFunc("/out", p.out)

	p.mux.HandleFunc("/chat", p.chat)

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
	msgs  []msg
	loc   sync.RWMutex
	tpl   *template.Template
}

type msg struct {
	when time.Time
	who  string
	what string
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
			Peers []peer
			Files map[string]string
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

	// TODO: Super cool idea. We have all download requests query our own local server.
	// Local server checks for all peers who have that file, and splits the request across them.
	// this would be pretty neat

	// Store the data in memory of the server.
	p.loc.Lock()
	p.mem[handler.Filename] = bits
	p.all.Files[handler.Filename] = "http://" + p.net.Address + "/get?file=" + handler.Filename
	data := net.EncodeFileList(p.all)
	p.loc.Unlock()
	p.net.Send(data)

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
func (p *Portal) list() map[string]string {
	p.loc.RLock()
	files := map[string]string{}
	for key, val := range p.all.Files {
		files[key] = val
	}
	for key := range p.mem {
		files[key] = "/get?file=" + key
	}
	p.loc.RUnlock()
	return files
}

// another short term hack
func (p *Portal) peerslist() []peer {
	p.loc.RLock()
	names := make([]peer, 0, len(p.peers))
	for _, pp := range p.peers {
		if pp.Alive {
			names = append(names, pp)
		}
	}
	p.loc.RUnlock()
	// sort.Strings(names)
	return names
}

func (p *Portal) chat(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		decode := json.NewDecoder(r.Body)
		data := map[string]string{}
		err := decode.Decode(&data)
		if err != nil {
			log.Printf("failed to decode send: %s", err)
		}
		r.Body.Close()
		m := msg{
			who:  data["who"],
			what: data["msg"],
			when: time.Now(),
		}
		p.msgs = append(p.msgs, m)
		p.net.Send(net.EncodeChat(m.who, m.what))
	} else if r.Method == http.MethodGet {
		v := r.URL.Query().Get("t")
		var num int
		if v != "" {
			num, _ = strconv.Atoi(v)
		}
		for i, msg := range p.msgs {
			if i >= num {
				fmt.Fprintf(w, chatTPL, msg.when.Format(timeFmt), msg.who, msg.what)
			}
		}
	}
}

func (p *Portal) out(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		p.msgs = append(p.msgs, msg{
			who:  r.Form.Get("who"),
			what: r.Form.Get("msg"),
			when: time.Now(),
		})
		p.net.Send(net.EncodeChat(r.Form.Get("who"), r.Form.Get("msg")))
	}
	http.Redirect(w, r, "/msg", http.StatusSeeOther)
}

var chatTPL = `<p class="msg"><span class="when">%s</span><span class="who">%s:</span><span class="what">%s</span></p>`

var timeFmt = "Jan _2 15:04:05"

func (p *Portal) msg(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")

	fmt.Fprint(w, `<link rel="stylesheet" href="/style.css">`+"\n"+`<div class="chats">`+"\n")
	var last int
	for i, msg := range p.msgs {
		fmt.Fprintf(w, chatTPL, msg.when.Format(timeFmt), msg.who, msg.what)
		last = i + 1
	}

	// Long pull http response of remaining content
	if f, ok := w.(http.Flusher); ok {
		ticker := time.NewTicker(time.Millisecond * 500) // twice a second
		var err error
		for i := 0; err == nil; i++ {
			// Terrible way of doing this
			if tail := len(p.msgs); last < tail {
				for j := last; j < tail; j++ {
					msg := p.msgs[j]
					fmt.Fprintf(w, chatTPL, msg.when.Format(timeFmt), msg.who, msg.what)
					last = j + 1
				}
			}

			// Send a ticker
			if _, err = fmt.Fprintf(w, `<span style="display:none">%d</span>`+"\n", i); err == nil { // style="display:none"
				f.Flush()
				<-ticker.C
			}
		}
		ticker.Stop()
		log.Printf("client socket closed")
	}
}
