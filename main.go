package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/bign8/myLANta/net"
	"github.com/bign8/myLANta/web"
)

var portz = flag.String("port", "9696", "port to serve on")

func main() {
	exit := make(chan int, 10)
	log.Printf("Launching Server.")
	network := net.New(exit)

	flag.Parse()
	log.Println("Serving on :" + *portz)
	go func() {
		panic(http.ListenAndServe(":"+*portz, web.New(network)))
	}()

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, os.Interrupt)

	go func() {
		for {
			sendHeartbeat(network)
			time.Sleep(time.Second * 5)
		}
	}()

	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				panic("stdin blew up: " + err.Error())
			}
			if n == 1 {
				switch buf[0] {
				case 'h':
					sendHeartbeat(network)
				case 'c':
					log.Printf("Current Clients: %#v", network.Clients())
				}
			}
		}
	}()

	<-cancel
	close(exit)

	log.Printf("goodbye")
}

func sendHeartbeat(network *net.Network) {
	hb := net.Heartbeat{
		Clients: network.Clients(),
		Files:   map[string]string{},
	}
	msg, err := json.Marshal(hb)
	if err != nil {
		log.Printf("this aint working out.")
		panic(err)
	}
	bytes := []byte{0, 0}
	binary.LittleEndian.PutUint16(bytes, uint16(len(msg)))
	network.Outgoing <- &net.Message{
		Target: 0,
		Raw:    append(bytes, msg...),
	}
}
