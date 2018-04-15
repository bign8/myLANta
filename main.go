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

var port = flag.String("port", "9696", "port to serve on")

func main() {
	flag.Parse()
	log.Println("Serving on :" + *port)

	exit := make(chan int, 10)
	network := net.New(*port, exit)
	go func() {
		panic(http.ListenAndServe(":"+*port, web.New(network)))
	}()

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, os.Interrupt)

	go networkController(network)
	go consoleDebugger(network)

	<-cancel
	close(exit)

	log.Printf("goodbye")
}

// This could go somewhere else, not sure yet good design for this.
func networkController(n *net.Network) {
	for {
		msg := <-n.Incoming
		switch msg.Kind {
		case net.MsgKindPing:
			n.SendHeartbeat() // maybe controller handles this.
		case net.MsgKindHeartbeat:
			// nothing to do i guess
		case net.MsgKindChat:
			// I dont like this.. maybe we should make a channel for each message type instead of a single one.
			chat := net.DecodeChat(msg)
			log.Printf("Got Chat: %q", chat.Text)
		case net.MsgKindFiles:
			fl := net.DecodeFileList(msg)
			// TODO: set on web
			log.Printf("Got Files: %#v", fl.Files)
		}
	}

}

func consoleDebugger(network *net.Network) {
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
		case 'l': // l = list peer
			log.Printf("Current Peers: %#v", network.Peers())
		}
	}
}
