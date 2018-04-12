package main

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	exit := make(chan int, 10)
	log.Printf("Launching Server.")
	network := RunServer(exit)

	go func() {
		hb := Heartbeat{
			Clients: []string{"a client"},
			Files:   map[string]string{"afile": "jajajaja"},
		}

		for {
			msg, err := json.Marshal(hb)
			if err != nil {
				log.Printf("this aint working out.")
				panic(err)
			}
			bytes := []byte{0, 0}
			binary.LittleEndian.PutUint16(bytes, uint16(len(msg)))
			log.Printf("len: %#v   :::    msg: %#v", bytes, msg)
			time.Sleep(time.Second)
			network.outgoing <- &Message{
				Target: BroadcastTarget,
				Raw:    append(bytes, msg...),
			}
		}
	}()

	buf := make([]byte, 1)
	os.Stdin.Read(buf)
	exit <- 1
	exit <- 1
	log.Printf("goodbye")
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
