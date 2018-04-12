package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/bign8/myLANta/mylanta"
	"github.com/bign8/myLANta/web"
)

var portz = flag.String("PORT", "8080", "port to serve on")

func main() {
	exit := make(chan int, 10)
	log.Printf("Launching Server.")
	network := mylanta.RunServer(exit)

	flag.Parse()
	log.Println("Serving on :" + *portz)
	go func() {
		panic(http.ListenAndServe(":"+*portz, web.New(network)))
	}()

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, os.Interrupt)

	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				panic("stdin blew up: " + err.Error())
			}
			if n == 1 {
				switch buf[0] {
				case 'p':
					sendHeartbeat(network)
				}
			}
		}
	}()

	<-cancel
	close(exit)

	log.Printf("goodbye")
}

func sendHeartbeat(network *mylanta.Network) {
	clients := []string{}
	for _, c := range network.Connections[1:] {
		if c.Addr == nil {
			break
		}
		clients = append(clients, c.Addr.String())
	}
	hb := mylanta.Heartbeat{
		Clients: clients,
		Files:   map[string]string{},
	}
	msg, err := json.Marshal(hb)
	if err != nil {
		log.Printf("this aint working out.")
		panic(err)
	}
	bytes := []byte{0, 0}
	binary.LittleEndian.PutUint16(bytes, uint16(len(msg)))
	network.Outgoing <- &mylanta.Message{
		Target: mylanta.BroadcastTarget,
		Raw:    append(bytes, msg...),
	}
}
