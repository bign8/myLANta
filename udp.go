package main

import (
	"encoding/binary"
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
	connections []Client
	connLookup  map[string]int16
	lastID      int32
	outgoing    chan *Message
}

func RunServer(exit chan int) *Network {
	network := &Network{
		connections: make([]Client, maxClient), // max of int16
		connLookup:  map[string]int16{},
		outgoing:    make(chan *Message, 100),
		lastID:      1,
	}
	rand.Seed(time.Now().Unix())
	castport := strconv.Itoa(rand.Intn(65535-49152) + 49152) //49152 to 65535

	var err error
	network.connections[0].Addr, err = net.ResolveUDPAddr("udp", discoveryAddr)
	if err != nil {
		panic(err)
	}
	network.connections[1].Addr, err = net.ResolveUDPAddr("udp", ":"+castport)
	if err != nil {
		panic(err)
	}
	network.conn, err = net.ListenUDP("udp", network.connections[1].Addr)
	if err != nil {
		panic(err)
	}
	network.bconn, err = net.ListenMulticastUDP("udp", nil, network.connections[0].Addr)
	if err != nil {
		panic(err)
	}
	log.Printf("I am %s", network.connections[1].Addr.String())
	go runBroadcastListener(network, exit)
	return network
}

func runBroadcastListener(s *Network, exit chan int) {
	log.Printf("Online.")
	incoming := make(chan *Message, 100)
	go s.listen(s.conn, s.connections[1].Addr.Port, incoming)
	go s.listen(s.bconn, s.connections[1].Addr.Port, incoming)

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
			log.Printf("RESULTING DATA: %s", result.Data)
		case msg := <-s.outgoing:
			if msg.Target > int16(atomic.LoadInt32(&s.lastID)) {
				break // can't find this user
			}
			addr := s.connections[msg.Target].Addr
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
			val := atomic.AddInt32(&s.lastID, 1)
			if val > maxClient {
				panic("too many clients have connected")
			}
			log.Printf("  New conn, assigning idx: %d.", val)
			connidx = int16(val)
			s.connLookup[addr] = connidx
			s.connections[connidx] = Client{Addr: ipaddr, ID: connidx, Alive: true}
		}
		incoming <- &Message{
			Raw:    buf[:n],
			Target: connidx,
		}
	}
}
