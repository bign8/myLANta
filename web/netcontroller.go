package web

import (
	"context"
	"log"
	stdnet "net"
	"time"

	"github.com/bign8/myLANta/net"
)

// Peer is all the metadata for a peer.
type peer struct {
	Addr     *stdnet.UDPAddr
	ID       int16
	Alive    bool
	LastPing time.Time
	Name     string
}

// This could go somewhere else, not sure yet good design for this.
func networkController(ctx context.Context, n *net.Network, p *Portal) {

	go func() {
		// TODO: find better spot for this... this will do but it should be better
		for ctx.Err() == nil {
			time.Sleep(time.Second * 5)                 // TODO: parameterize this
			expired := time.Now().Add(-time.Second * 5) // TODO(bign8): parameterize
			p.loc.Lock()
			for key, peer := range p.peers {
				if peer.Alive && peer.LastPing.Before(expired) {
					peer.Alive = false
					p.peers[key] = peer
				}
			}
			p.loc.Unlock()
		}
	}()

	for ctx.Err() == nil {
		incoming := <-n.Incoming

		// Update peers
		addr := incoming.Addr.String()
		p.loc.Lock()
		con, ok := p.peers[addr]
		if !ok {
			// setup new peer here.
			con.Addr = incoming.Addr
			con.Name = "Unknown" // TODO: store name?
		}
		con.Alive = true
		con.LastPing = time.Now()
		p.peers[addr] = con
		p.loc.Unlock()

		switch incoming.Kind {
		case net.MsgKindPing:
			n.Send(net.NewMsgHeartbeat()) // maybe controller handles this.
		case net.MsgKindHeartbeat:
			// nothing to do i guess
		case net.MsgKindChat:
			// I dont like this.. maybe we should make a channel for each message type instead of a single one.
			chat := net.DecodeChat(incoming)
			p.msgs = append(p.msgs, msg{
				who:  chat.Name,
				what: chat.Text,
				when: time.Now(),
			})
			log.Printf("Got Chat: %q", chat.Text)
		case net.MsgKindFiles:
			fl := net.DecodeFileList(incoming)
			// TODO: set on web
			log.Printf("Got Files: %#v", fl.Files)
		}
	}

}
