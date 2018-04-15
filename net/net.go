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
	port string // port this network is running on

	// Internal, but thread safe members
	outbox chan *sending // outgoing messages queued by Send

	// Public Interface to Network
	Incoming chan *Message // Messages sent to this will be re-emitted for consumption by controller
}

func chk(msg string, err error) {
	if err != nil {
		log.Fatal(msg, err)
	}
}

// New creates a new network.
func New(port string) *Network {
	return &Network{
		port:     port,
		outbox:   make(chan *sending, 100),
		Incoming: make(chan *Message, 100),
	}
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

type sending struct {
	data []byte
	addr *net.UDPAddr
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
// This should be the only method that accesses peers.
func (n *Network) Run(ctx context.Context) error {
	incoming := make(chan inbound, 100)
	lookup := map[string]bool{} // lookup the address so we can ignore our own broadcasts

	// Setups peers 0 and 1 to be multicast and self
	baddr, err := net.ResolveUDPAddr("udp", discoveryAddr)
	chk("ResolveUDPAddr Multicast", err)
	lookup[baddr.String()] = true

	addr, err := net.ResolveUDPAddr("udp", n.port)
	chk("ResolveUDPAddr Self", err)
	lookup[addr.String()] = true

	// Setting loopbacks for my address (this is so we don't send messages to ourself)
	ips := getMyIPs()
	log.Printf("My IPs: %s", ips)
	for _, ip := range ips {
		lookup[ip+n.port] = true
	}

	// Listen to multicast
	multicast, err := net.ListenMulticastUDP("udp", nil, baddr)
	if err != nil {
		return err
	}
	go listen(multicast, incoming)

	// Listen to directed udp messages
	direct, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	go listen(direct, incoming)

	// notify everyone that we are starting up
	go n.Send(NewMsgPing())

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	for {
		select {
		case msg := <-incoming:
			_, ok := lookup[msg.from.String()]
			if ok {
				continue // ignore my own messages
			}
			// receive message
			n.Incoming <- &Message{
				Kind: MsgKind(msg.data[0]),
				Data: msg.data[1:],
				Addr: msg.from,
			}
		case letter := <-n.outbox:
			to := letter.addr
			if to == nil {
				to = baddr // default to broadcast
			}
			n, err := direct.WriteToUDP(letter.data, to)
			if err != nil {
				fmt.Println("Error: ", err, " Bytes Written: ", n)
			}
			letter.done <- err
		case <-ctx.Done():
			fmt.Println("Killing UDP Server")
			// TODO: shutdown all conns
			return direct.Close()
		case <-ticker.C:
			go n.Send(NewMsgHeartbeat())
		}
	}
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
