package net

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var discoveryAddr = "239.1.12.123:9999"

const maxClient = 1 << 16

// Network is a udp network manager.
type Network struct {
	conn        *net.UDPConn
	bconn       *net.UDPConn
	Connections []Client
	connLookup  *sync.Map
	lastID      int32
	Outgoing    chan *Message
	myips       []string
	mahport     string
}

// Clients gives the active clients list
func (n *Network) Clients() []Client {
	result := make([]Client, 0, int(atomic.LoadInt32(&n.lastID))+1)
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
		Connections: make([]Client, maxClient), // max of int16
		connLookup:  &sync.Map{},
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
			network.connLookup.Store(sliceaddr+":"+network.mahport, int16(1))
		}
	}
	log.Printf("My IPs: %s", network.myips)
	log.Printf("I am %s", network.Connections[1].Addr.String())
	go runBroadcastListener(network, exit)
	return network
}

func runBroadcastListener(n *Network, exit chan int) {
	log.Printf("Online.")
	incoming := make(chan *Message, 100)
	go n.listen(n.conn, n.Connections[1].Addr.Port, incoming)
	go n.listen(n.bconn, n.Connections[1].Addr.Port, incoming)

	alive := true
	for alive {
		timeout := time.After(time.Second * 15)
		select {
		case msg := <-incoming:
			con := n.Connections[msg.Target]
			con.Alive = true
			con.LastPing = time.Now()
			n.Connections[msg.Target] = con
			length := binary.LittleEndian.Uint16(msg.Raw[:2])
			if length > 1500 {
				panic("TOO BIG MSG")
			}
			n.processPeers(decode(msg, length))
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

func (n *Network) processPeers(result *Message) {
	for idx, peer := range result.Data.Clients {
		if idx == 0 {
			// hax, i know the first user is the person who sent it...
			n.Connections[result.Target].Name = peer.Name
			continue
		}
		if _, ok := n.connLookup.Load(peer.Addr.String()); !ok {
			n.connLookup.Store(peer.Addr.String(), n.addConn(peer.Name, peer.Addr.String(), nil))
		}
	}

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
		if now.Sub(c.LastPing) > time.Second*30 {
			log.Printf("   timed out: %#v", c)
			c.Alive = false
			n.Connections[i] = c
		}
	}
}

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
	for _, maddr := range n.myips {
		raddr := maddr + ":" + n.mahport
		if addr == raddr {
			log.Printf("   Ignoring peer %s", addr)
			return 1 // ignore my own messages
		}
	}
	log.Printf("  New conn (%s), assigning idx: %d.", addr, val)
	n.connLookup.Store(addr, int16(val))
	n.Connections[val] = Client{Addr: ipaddr, ID: int16(val), Alive: true, Name: name}
	return int16(val)
}

func (n *Network) listen(conn *net.UDPConn, me int, incoming chan *Message) {
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
		var connidx int16
		lv, ok := n.connLookup.Load(addr)
		if !ok {
			connidx = n.addConn("", addr, ipaddr)
		} else {
			connidx = lv.(int16)
		}
		incoming <- &Message{
			Raw:    buf[:m],
			Target: connidx,
		}
	}
}

// Client is all the metadata for a client.
type Client struct {
	Addr     *net.UDPAddr
	ID       int16 `js:"-"`
	Alive    bool  `js:"-"`
	LastPing time.Time
	Name     string
}

// Message containing the messaging.
type Message struct {
	Raw    []byte
	Target int16
	Data   Heartbeat
}

func decode(m *Message, length uint16) *Message {
	dcd := &Message{
		Raw:    m.Raw,
		Target: m.Target,
	}
	hb := Heartbeat{}
	lol := json.Unmarshal(m.Raw[2:], &hb)
	if lol != nil {
		panic(lol)
	}
	dcd.Data = hb
	return dcd
}

// Heartbeat information.
type Heartbeat struct {
	Clients []Client
	Files   map[string]string // map of file name to md5
}
