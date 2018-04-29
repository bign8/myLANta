package svc

import (
	"time"

	"github.com/bign8/myLANta/model"
)

type peer struct {
	name  string    // human readable
	addr  string    // machine readable
	seen  time.Time // last time this person was seen
	files []string  // array of hashes
}

func (svc *Service) touch(addr string) {
	p, ok := svc.peers[addr]
	if !ok {
		p = &peer{
			name: "TODO: generate fun initial name",
			addr: addr,
		}
		svc.peers[addr] = p
	}
	p.seen = time.Now()
}

// Peers provides a map from address to a useful name.
func (svc *Service) Peers() map[string]string {
	svc.mux.RLock()
	out := make(map[string]string, len(svc.peers))
	for k, v := range svc.peers {
		out[k] = v.name
	}
	svc.mux.RUnlock()
	return out
}

// Title gets or sets ones own title.
func (svc *Service) Title(in string) string {
	return in
}

// Listen allows streaming of messaging events.
// TODO(bign8): add historic paging through data
func (svc *Service) Listen() ([]*model.Message, <-chan *model.Message, func()) {
	return nil, nil, nil
}

// Message sends a message to an individual.
func (svc *Service) Message(who, what string) error {
	data := []byte{msgChat}
	svc.outbox <- &model.Message{Addr: "", Data: data}
	return nil
}
