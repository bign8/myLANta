package web

import (
	"net/http"

	"github.com/bign8/myLANta/mylanta"
)

// New constructs a new web portan handler.
func New(net *mylanta.Network) http.Handler {
	p := &Portal{
		mux: http.NewServeMux(),
		web: http.FileServer(http.Dir("web")),
		net: net,
	}
	p.mux.Handle("/", p.web)
	p.mux.HandleFunc("/add", p.add)
	p.mux.HandleFunc("/get", p.get)
	return p.mux
}

// Portal is the web portal driver.
type Portal struct {
	mux *http.ServeMux
	web http.Handler
	net *mylanta.Network
}

func (p *Portal) add(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "TODO", http.StatusNotImplemented)
}

func (p *Portal) get(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "TODO", http.StatusNotImplemented)
}
