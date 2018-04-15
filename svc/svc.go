package svc

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/bign8/myLANta/model"
	"github.com/bign8/myLANta/net"
)

var _ model.MyLANta = (*Service)(nil)

// New constructs a MyLANta service.
func New() *Service {
	svc := &Service{
		peers: make(map[string]*peer),
		files: make(map[string]*file),
	}
	http.HandleFunc("/dl", svc.dl)
	return svc
}

// Service provides a data abstraction between p2p networking and any UI.
type Service struct {
	req chan request

	peers map[string]*peer // key: addr (TODO: move to run below)
	files map[string]*file // key: hash (TODO: move to run below)
	mux   sync.RWMutex
}

type request struct {
	kind byte
	data interface{}
	done chan<- request
}

// Run connects this service to a network transport tier.
func (svc *Service) Run(ctx context.Context, inbox <-chan *model.Message, outbox chan<- *model.Message) error {
	var ( // variables governed by this routine
		peers = make(map[string]*peer) // key: addr
		files = make(map[string]*file) // key: hash
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case m := <-inbox:
			svc.touch(m.Addr)
			switch net.MsgKind(m.Data[0]) {
			case net.MsgKindChat:
				log.Printf("web got chat: %q %v %v", string(m.Data), peers, files)
			case net.MsgKindFiles:
				log.Printf("web got files from %q", m.Addr)
			case net.MsgKindHeartbeat:
				log.Printf("web got beat from %q", m.Addr)
			case net.MsgKindPing:
				log.Printf("web got ping from %q", m.Addr)
				outbox <- &model.Message{Data: []byte{byte(net.MsgKindHeartbeat)}}
			}

		case r := <-svc.req:
			switch r.kind {
			case '?':
				r.done <- request{ /* help me plz */ }
			default:
				r.done <- request{data: errors.New("TODO")}
			}
		}
	}
}
