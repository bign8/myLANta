package net

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"
)

var discoveryAddr = "239.1.12.123:9999"

const maxClient = int16(1<<15 - 1)

// Network is a udp network manager.
// TODO: rename this, possibly 'net.Interface' or 'net.Conn' not sure yet. 'net.Manager' is such a generic name... but maybe it works here.
type Network struct {

	// Private members that should only be accessed by Run
	// TODO: move these off the network object
	lookup map[string]int16 // only used by the 'listen' goroutine
	peers  []peer           // list of all peers (only Run touches this list)

	// Internal, but thread safe members
	outbox chan *sending // outgoing messages queued by Send

	// Public Interface to Network
	Incoming chan *Message // Messages sent to this will be re-emitted for consumption by controller
}

// Peer is all the metadata for a peer.
type peer struct {
	Addr     *net.UDPAddr
	ID       int16
	Alive    bool
	LastPing time.Time
	Name     string
}

func chk(msg string, err error) {
	if err != nil {
		log.Fatal(msg, err)
	}
}

// New creates a new network.
func New(port string) *Network {
	n := &Network{
		peers:    make([]peer, 0, maxClient), // max of int16
		lookup:   map[string]int16{},
		outbox:   make(chan *sending, 100),
		Incoming: make(chan *Message, 100),
	}

	// Setting loopbacks for my address (this is so we don't send messages to ourself)
	ips := getMyIPs()
	log.Printf("My IPs: %s", ips)
	for _, ip := range ips {
		n.lookup[ip+port] = 1
	}

	// Setups peers 0 and 1 to be multicast and self
	addr, err := net.ResolveUDPAddr("udp", discoveryAddr)
	chk("ResolveUDPAddr Multicast", err)
	n.addPeer("multi", addr) // peer 0
	addr, err = net.ResolveUDPAddr("udp", port)
	chk("ResolveUDPAddr Self", err)
	n.addPeer("self", addr) // peer 1

	return n
}

func getMyIPs() (mine []string) {
	itfs, err := net.Interfaces()
	chk("net.Interfaces", err)

	for _, itf := range itfs {
		switch {
		case itf.Flags&net.FlagUp != net.FlagUp:
			continue // skip down interfaces
		case itf.Flags&net.FlagLoopback == net.FlagLoopback:
			continue // skip loopbacks
		case itf.HardwareAddr == nil:
			continue // not real network hardware
		}
		if multiz, err := itf.MulticastAddrs(); err != nil {
			log.Fatal("cant get the IPs  MulticastAddress", err)
		} else if len(multiz) == 0 {
			continue // no multicast
		}

		addrs, err := itf.Addrs()
		if err != nil {
			log.Fatal("cant get the IPs Addrs", err)
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			chk("ParseCIDR", err)

			ipv4 := ip.To4()
			if ipv4 == nil {
				continue // skip ipv4 addrs
			}
			mine = append(mine, ipv4.String())
		}
	}
	return mine
}

// Network Public Functions

// Peers gives the active peers list
func (n *Network) Peers() []string {
	out := make([]string, 0)
	for key, val := range n.lookup {
		if val > 1 {
			out = append(out, key)
		}
	}
	return out
}

type sending struct {
	data []byte
	addr string
	done chan<- error
}

// Send broadcasts a message to it's indended consumer.
func (n *Network) Send(msg *Message) error {
	if len(msg.Data) > 1500 {
		return errors.New("net.Send: message to large")
	}
	done := make(chan error, 1)
	n.outbox <- &sending{
		data: append([]byte{byte(msg.Kind)}, msg.Data...),
		addr: msg.Addr,
		done: done,
	}
	return <-done
}

// Run will loop over messages from the network and decide what to do with them.
// This will fire of a 'listen' goroutine for the multicast network connection.
// This should be the only method that accesses peers or lookup.
func (n *Network) Run(ctx context.Context) error {
	incoming := make(chan inbound, 100)

	// Listen to multicast
	multicast, err := net.ListenMulticastUDP("udp", nil, n.peers[0].Addr)
	if err != nil {
		return err
	}
	go listen(multicast, incoming)

	// Listen to directed udp messages
	direct, err := net.ListenUDP("udp", n.peers[1].Addr)
	if err != nil {
		return err
	}
	go listen(direct, incoming)

	// notify everyone that we are starting up
	go n.Send(NewMsgPing())

	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()
	for {
		select {
		case msg := <-incoming:
			idx, ok := n.lookup[msg.from.String()]
			if !ok {
				idx = n.addPeer("", msg.from)
			} else if idx == 1 {
				continue // ignore my own messages
			}

			// Update last seen timers
			con := n.peers[idx]
			con.Alive = true
			con.LastPing = time.Now()
			n.peers[idx] = con

			// receive message
			n.Incoming <- &Message{
				Kind: MsgKind(msg.data[0]),
				Data: msg.data[1:],
				Addr: msg.from.String(),
			}
		case letter := <-n.outbox:
			idx, ok := n.lookup[letter.addr]
			if !ok {
				letter.done <- errors.New("net.Send: cannot find address: " + letter.addr)
				continue
			}

			// send message
			addr := n.peers[idx].Addr
			n, err := direct.WriteToUDP(letter.data, addr)
			if err != nil {
				fmt.Println("Error: ", err, " Bytes Written: ", n)
			}
			letter.done <- err
		case <-ctx.Done():
			fmt.Println("Killing UDP Server")
			// TODO: shutdown all conns
			return direct.Close()
		case now := <-ticker.C:
			n.Send(NewMsgPing())

			// look for (and expire) stale peers.
			// TODO(lologorithm): currently mutates the peers list which I don't like.
			expired := now.Add(-time.Second * 35) // TODO(bign8): parameterize
			for i := 2; i < len(n.peers); i++ {
				if c := n.peers[i]; c.Alive && c.LastPing.Before(expired) {
					log.Printf("   timed out: %#v", c)
					c.Alive = false
					n.peers[i] = c
				}
			}
		}
	}
}

// addPeer will add a connection to the peers map and slice.
// since this function touches peers only use in Run goroutine.
func (n *Network) addPeer(name string, addr *net.UDPAddr) int16 {
	val := int16(len(n.peers))
	if val > maxClient {
		panic("too many clients have connected")
	}
	log.Printf("  New conn (%s), assigning idx: %d.", addr.String(), val)
	n.lookup[addr.String()] = val

	n.peers = append(n.peers, peer{
		Addr:     addr,
		ID:       val,
		Alive:    true,
		Name:     name,
		LastPing: time.Now(),
	})
	return val
}

type inbound struct {
	from *net.UDPAddr
	data []byte
}

// listen will listen to the given conn.
func listen(conn *net.UDPConn, incoming chan<- inbound) {
	buf := make([]byte, 2048)
	for {
		m, from, err := conn.ReadFromUDP(buf)
		chk("ReadFromUDP", err)
		if m == 0 {
			continue
		}
		incoming <- inbound{from: from, data: buf[:m]}
	}
}
