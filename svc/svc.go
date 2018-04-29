package svc

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bign8/myLANta/model"
)

const (
	msgPing = byte(iota) // welcome to the new application
	msgPong              // initial response from all peers, their peers and file lists
	msgChat              // someone sends a message
	msgFile              // someones file listing has changed
	msgBeep              // healthcheck broadcasts
)

var (
	_ model.MyLANta = (*Service)(nil)

	ttl  = flag.Duration("ttl", 30*time.Second, "time for peers to die after no health")
	beat = flag.Duration("del", 5*time.Second, "delay between heartbeats")
)

// New constructs a MyLANta service.
func New(outbox chan<- *model.Message, inbox <-chan *model.Message) *Service {
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

	outbox chan<- *model.Message
	inbox  <-chan *model.Message
}

type request struct {
	kind byte
	data interface{}
	done chan<- request
}

// Run connects this service to a network transport tier.
func (svc *Service) Run(ctx context.Context) error {
	var ( // variables governed by this routine
		peers = make(map[string]*peer) // key: addr
		files = make(map[string]*file) // key: hash
		heart = time.NewTicker(*beat)
	)

	for {
		select {
		case <-ctx.Done():
			heart.Stop()
			return ctx.Err()

		case m := <-svc.inbox:
			svc.touch(m.Addr)
			switch m.Data[0] {
			case msgPing:
				svc.outbox <- &model.Message{Addr: m.Addr, Data: []byte{msgPong}} // TODO: send ping data
			case msgPong:
				// TODO: load pong data
			case msgChat:
				log.Printf("web got chat: %q %v %v", string(m.Data), peers, files)
				// TODO: send to all listeners (in non-blocking fashion)
			case msgFile:
				// TODO: update and emit event to listeners
			case msgBeep: // don't care, just needed to touch site
			default:
				panic("unsupported type")
			}

		case r := <-svc.req:
			switch r.kind {
			case '?':
				r.done <- request{ /* help me plz */ }
			default:
				r.done <- request{data: errors.New("TODO")}
			}

		case <-heart.C:
			svc.outbox <- &model.Message{Data: []byte{msgBeep}}
		}
	}
}
