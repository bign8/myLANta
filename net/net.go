package net

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var discoveryAddr = "239.1.12.123:9999"

const maxClient = 1 << 16

// Network is a udp network manager.
type Network struct {
	conn        *net.UDPConn
	bconn       *net.UDPConn
	Connections []Peer
	connLookup  map[string]int16 // only used by the 'listen' goroutine
	lastID      int32
	Outgoing    chan *Message
	myips       []string
	mahport     string
}

// Client is all the metadata for a peer.
type Peer struct {
	Addr     *net.UDPAddr
	ID       int16 `js:"-"`
	Alive    bool  `js:"-"`
	LastPing time.Time
	Name     string
}

// Clients gives the active clients list
func (n *Network) Clients() []Peer {
	result := make([]Peer, 0, int(atomic.LoadInt32(&n.lastID))+1)
	for _, c := range n.Connections[1:] {
		if c.Addr != nil && c.Alive {
			result = append(result, c)
		}
	}
	return result
}

// New creates a new network.
func New(exit chan int) *Network {
	network := &Network{
		Connections: make([]Peer, maxClient), // max of int16
		connLookup:  map[string]int16{},
		Outgoing:    make(chan *Message, 100),
		lastID:      1,
	}
	rand.Seed(time.Now().Unix())
	network.mahport = strconv.Itoa(rand.Intn(65535-49152) + 49152) //49152 to 65535

	var err error
	network.Connections[0].Addr, err = net.ResolveUDPAddr("udp", discoveryAddr)
	if err != nil {
		panic(err)
	}
	network.Connections[1].Addr, err = net.ResolveUDPAddr("udp", ":"+network.mahport)
	if err != nil {
		panic(err)
	}
	network.Connections[1].Name, err = os.Hostname()
	if err != nil {
		network.Connections[1].Name = err.Error()
	}
	network.Connections[1].ID = 1
	network.conn, err = net.ListenUDP("udp", network.Connections[1].Addr)
	if err != nil {
		panic(err)
	}
	network.bconn, err = net.ListenMulticastUDP("udp", nil, network.Connections[0].Addr)
	if err != nil {
		panic(err)
	}
	itfs, err := net.Interfaces()
	if err != nil {
		panic("cant get the IPs")
	}
	for _, itf := range itfs {
		addrs, err := itf.Addrs()
		if err != nil {
			panic("cant get the IPs")
		}
		for _, addr := range addrs {
			sliceaddr := strings.Split(addr.String(), "/")[0]
			network.myips = append(network.myips, sliceaddr)
			network.connLookup[sliceaddr+":"+network.mahport] = 1
		}
	}
	log.Printf("My IPs: %s", network.myips)
	log.Printf("I am %s", network.Connections[1].Addr.String())
	go runBroadcastListener(network, exit)
	go heartbeater(network, exit)
	return network
}

func heartbeater(n *Network, exit chan int) {
	for {
		timer := time.After(time.Second * 10)
		select {
		case <-timer:
			n.SendPing()
		case _, ok := <-exit:
			if ok {
				panic("why did we get OK from closed exit")
			}
			return
		}
	}
}

func runBroadcastListener(n *Network, exit chan int) {
	log.Printf("Online.")
	incoming := make(chan *Message, 100)
	go n.listen(n.conn, incoming)
	go n.listen(n.bconn, incoming)

	alive := true
	for alive {
		timeout := time.After(time.Second * 15)
		select {
		case msg := <-incoming:
			con := n.Connections[msg.Target]
			con.Alive = true
			con.LastPing = time.Now()
			length := binary.LittleEndian.Uint16(msg.Raw[1:3])
			if length > 1500 {
				panic("TOO BIG MSG")
			}
			log.Printf("New message from: %#v", con.Addr)
			switch msg.Kind {
			case MsgKindPing:
				n.SendHeartbeat()
			case MsgKindHeartbeat:
				// nothing to do i guess
			case MsgKindChat:
				chat := decodeChat(msg)
				log.Printf("Got Chat: %s", chat.Text)
			case MsgKindFiles:
			}
		case msg := <-n.Outgoing:
			if msg.Target > int16(atomic.LoadInt32(&n.lastID)) {
				break // can't find this user
			}
			addr := n.Connections[msg.Target].Addr
			if n, err := n.conn.WriteToUDP(msg.Raw, addr); err != nil {
				fmt.Println("Error: ", err, " Bytes Written: ", n)
			}
		case <-exit:
			alive = false
			break
		case <-timeout:
		}
		n.timeoutStale()
	}
	fmt.Println("Killing Socket Server")
	n.conn.Close()
}

func (n *Network) timeoutStale() {
	now := time.Now()
	for i := range n.Connections {
		if i < 2 {
			continue
		}
		c := n.Connections[i]
		if c.Addr == nil {
			break
		}
		if !c.Alive {
			continue
		}
		if now.Sub(c.LastPing) > time.Second*35 {
			log.Printf("   timed out: %#v", c)
			c.Alive = false
			n.Connections[i] = c
		}
	}
}

// addConn will add a connection to the connections map and slice.
// since this function touches connLookup only use in listen goroutine.
func (n *Network) addConn(name string, addr string, ipaddr *net.UDPAddr) int16 {
	val := atomic.AddInt32(&n.lastID, 1)
	if val > maxClient {
		panic("too many clients have connected")
	}
	if ipaddr == nil {
		var err error
		ipaddr, err = net.ResolveUDPAddr("udp", addr)
		if err != nil {
			panic("bad client addr")
		}
	}
	log.Printf("  New conn (%s), assigning idx: %d.", addr, val)
	n.connLookup[addr] = int16(val)
	n.Connections[val] = Peer{Addr: ipaddr, ID: int16(val), Alive: true, Name: name, LastPing: time.Now()}
	return int16(val)
}

func (n *Network) listen(conn *net.UDPConn, incoming chan *Message) {
	buf := make([]byte, 2048)
	for {
		m, ipaddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("ERROR: ", err)
			return
		}
		if m == 0 {
			continue
		}
		// Is this the fastest and simplest way to lookup unique connection?
		addr := ipaddr.String()
		connidx, ok := n.connLookup[addr]

		if !ok {
			shouldProcess := true // default to processing the message
			// Check to see if this is my IP address
			for _, maddr := range n.myips {
				raddr := maddr + ":" + n.mahport
				if addr == raddr {
					log.Printf("   Ignoring peer %s", addr)
					n.connLookup[addr] = 1 // Force this IP to link to my own local address
					shouldProcess = false  // dont process my own messages
					break
				}
			}
			if !shouldProcess {
				continue // skip to next network message
			}
			connidx = n.addConn("", addr, ipaddr)
		}
		if connidx == 1 {
			continue // ignore my own messages
		}
		log.Printf("   msg contents: %#v", buf[:m])
		incoming <- &Message{
			Kind:   MsgKind(buf[0]),
			Raw:    buf[:m],
			Target: connidx,
		}
	}
}

func (n *Network) SendChat(msg string) {
	bytes := []byte{byte(MsgKindChat), 0, 0}
	binary.LittleEndian.PutUint16(bytes[1:], uint16(len(msg)))
	bytes = append(bytes, []byte(msg)...)
	n.Outgoing <- &Message{
		Target: 0, // broadcast index
		Kind:   MsgKindChat,
		Raw:    bytes,
	}
}

func (n *Network) SendPing() {
	bytes := []byte{byte(MsgKindPing), 0, 0} // ping doesnt have any data
	n.Outgoing <- &Message{
		Target: 0, // broadcast index
		Kind:   MsgKindPing,
		Raw:    bytes,
	}
}

func (n *Network) SendHeartbeat() {
	bytes := []byte{byte(MsgKindHeartbeat), 0, 0} // hb doesnt have any data
	n.Outgoing <- &Message{
		Target: 0, // broadcast index
		Kind:   MsgKindHeartbeat,
		Raw:    bytes,
	}
}
