package web

import (
	"net/http"

	"github.com/bign8/myLANta"
)

func New(net *myLANta.Network) *Portal {
	p := &Portal{
		mux: http.NewServeMux(),
		web: http.FileServer(http.Dir("web")),
	}
	p.mux.Handle("/", p.web)
	http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "TODO", http.StatusNotImplemented)
	})
	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "TODO", http.StatusNotImplemented)
	})
	return p
}

type Portal struct {
	mux *http.ServeMux
	web http.Handler
}
