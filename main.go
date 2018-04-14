package main

import (
	"bufio"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"

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
		buf := bufio.NewReader(os.Stdin)

		for {
			line, _, err := buf.ReadLine()
			if err != nil {
				panic("stdin blew up: " + err.Error())
			}
			if len(line) == 0 {
				continue
			}
			switch line[0] {
			case 'c':
				network.SendChat(string(line[2:]))
			case 'h': // h = heartbeat
				network.SendHeartbeat()
			case 'p': // p = ping
				network.SendPing()
			case 'l': // l = list
				log.Printf("Current Clients: %#v", network.Clients())
			}
		}
	}()

	<-cancel
	close(exit)

	log.Printf("goodbye")
}
