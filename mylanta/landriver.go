package mylanta

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

var discoveryAddr = "239.1.12.123:9999"

const maxClient = 1 << 16
const BroadcastTarget = 0

type Network struct {
	conn        *net.UDPConn
	bconn       *net.UDPConn
	Connections []Client
	connLookup  map[string]int16
	lastID      int32
	Outgoing    chan *Message
}

func RunServer(exit chan int) *Network {
	network := &Network{
		Connections: make([]Client, maxClient), // max of int16
		connLookup:  map[string]int16{},
		Outgoing:    make(chan *Message, 100),
		lastID:      1,
	}
	rand.Seed(time.Now().Unix())
	castport := strconv.Itoa(rand.Intn(65535-49152) + 49152) //49152 to 65535

	var err error
	network.Connections[0].Addr, err = net.ResolveUDPAddr("udp", discoveryAddr)
	if err != nil {
		panic(err)
	}
	network.Connections[1].Addr, err = net.ResolveUDPAddr("udp", ":"+castport)
	if err != nil {
		panic(err)
	}
	network.conn, err = net.ListenUDP("udp", network.Connections[1].Addr)
	if err != nil {
		panic(err)
	}
	network.bconn, err = net.ListenMulticastUDP("udp", nil, network.Connections[0].Addr)
	if err != nil {
		panic(err)
	}
	log.Printf("I am %s", network.Connections[1].Addr.String())
	go runBroadcastListener(network, exit)
	return network
}

func runBroadcastListener(s *Network, exit chan int) {
	log.Printf("Online.")
	incoming := make(chan *Message, 100)
	go s.listen(s.conn, s.Connections[1].Addr.Port, incoming)
	go s.listen(s.bconn, s.Connections[1].Addr.Port, incoming)

	alive := true
	for alive {
		select {
		case msg := <-incoming:
			if msg.Target == 0 {
				log.Printf("Heard my own multicast come back at me.")
				break
			}

			log.Printf("Got a message from (%d): %#v", msg.Target, msg.Raw)
			length := binary.LittleEndian.Uint16(msg.Raw[:2])
			if length > 1500 {
				panic("TOO BIG MSG")
			}

			result := decode(msg, length)
			for _, a := range result.Data.Clients {
				found := false
				for _, c := range s.Connections {
					if c.Addr == nil {
						break
					}
					if c.Addr.String() == a {
						found = true
						break
					}
				}
				if !found {
					s.connLookup[a] = s.addConn(a, nil)
				}
			}
		case msg := <-s.Outgoing:
			if msg.Target > int16(atomic.LoadInt32(&s.lastID)) {
				break // can't find this user
			}
			addr := s.Connections[msg.Target].Addr
			if n, err := s.conn.WriteToUDP(msg.Raw, addr); err != nil {
				fmt.Println("Error: ", err, " Bytes Written: ", n)
			}
		case <-exit:
			alive = false
			break
		}
	}
	fmt.Println("Killing Socket Server")
	s.conn.Close()
}

func (s *Network) addConn(addr string, ipaddr *net.UDPAddr) int16 {
	val := atomic.AddInt32(&s.lastID, 1)
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
	log.Printf("  New conn, assigning idx: %d.", val)
	s.connLookup[addr] = int16(val)
	s.Connections[val] = Client{Addr: ipaddr, ID: int16(val), Alive: true}
	return int16(val)
}

func (s *Network) listen(conn *net.UDPConn, me int, incoming chan *Message) {
	buf := make([]byte, 2048)
	for {
		n, ipaddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("ERROR: ", err)
			return
		}
		if n == 0 {
			continue
		}
		if ipaddr.Port == me {
			continue // ignore my own messages
		}
		// Is this the fastest and simplest way to lookup unique connection?
		addr := ipaddr.String()
		log.Printf("Incoming from %s", addr)
		connidx, ok := s.connLookup[addr]
		if !ok {
			connidx = s.addConn(addr, ipaddr)
		}
		incoming <- &Message{
			Raw:    buf[:n],
			Target: connidx,
		}
	}
}

type Client struct {
	Addr  *net.UDPAddr
	ID    int16
	Alive bool
}

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

type Heartbeat struct {
	Clients []string
	Files   map[string]string // map of file name to md5
}
