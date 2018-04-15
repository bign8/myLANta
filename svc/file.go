package svc

import (
	"errors"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/bign8/myLANta/model"
)

type file struct {
	name  string   // human readable
	hash  string   // machine readable
	data  []byte   // TODO: iff the data is too large, put it on disk somewhere
	peers []string // TODO: array of addresses
}

// Files gives a map from MD5 to name of a file.
func (svc *Service) Files() map[string]string {
	svc.mux.RLock()
	out := make(map[string]string, len(svc.files))
	for k, v := range svc.files {
		out[k] = v.name
	}
	svc.mux.RUnlock()
	return out
}

type listAddrSeen []struct {
	addr string
	seen time.Time
}

func (las listAddrSeen) Len() int           { return len(las) }
func (las listAddrSeen) Swap(a, b int)      { las[a], las[b] = las[b], las[a] }
func (las listAddrSeen) Less(a, b int) bool { return las[a].seen.After(las[b].seen) }

// Fetch gathers a file from a peer.
func (svc *Service) Fetch(hash string) ([]byte, error) {
	// expired := time.Now().Add(-*ttl)
	svc.mux.RLock()
	file, ok := svc.files[hash]
	if !ok || len(file.peers) == 0 { // TODO: or the peers are really old
		svc.mux.RUnlock()
		return nil, model.ErrFileDNE
	}
	list := make(listAddrSeen, len(file.peers))
	for i, p := range file.peers {
		list[i].addr = p
		list[i].seen = svc.peers[p].seen
	}
	svc.mux.RUnlock()
	sort.Sort(list)
	res, err := http.Get(list[0].addr + "/dl?=" + hash)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

// Serve adds a file to the network pool of files.
func (svc *Service) Serve(name string, data []byte) error {
	// TODO: perform MD5 hash

	svc.mux.Lock()
	// TODO: insert into map
	svc.mux.Unlock()
	return errors.New("TODO")
}

func (svc *Service) dl(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "todo", http.StatusNotImplemented)
}
