package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/bign8/myLANta/net"
	"github.com/bign8/myLANta/web"
)

var iport = flag.Int("port", 9696, "port to serve on")

func main() {
	flag.Parse()
	port := fmt.Sprintf(":%d", *iport)

	// Start up new server context
	ctx, exit := context.WithCancel(context.Background())
	network := net.New(port)
	wserver := &http.Server{Addr: port, Handler: web.New(ctx, network)}
	wserver.RegisterOnShutdown(exit) // bind network to wserver shutdown

	// Execute application
	log.Println("Serving on " + port)
	go func() {
		log.Fatal(network.Run(ctx))
	}()
	go func() {
		log.Fatal(wserver.ListenAndServe())
	}()
	go consoleDebugger(ctx, network)

	// Full gracefull shutdown
	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, os.Interrupt)
	<-cancel
	fmt.Println() // new line for terminals that echo control sequences
	log.Println("shutting down...")
	killCtx, done := context.WithTimeout(context.Background(), time.Second*10)
	defer done()
	wserver.Shutdown(killCtx) // kills webserver
	log.Print("goodbye")
}

func consoleDebugger(ctx context.Context, network *net.Network) {
	buf := bufio.NewReader(os.Stdin)

	for ctx.Err() == nil {
		line, _, err := buf.ReadLine()
		if err != nil {
			panic("stdin blew up: " + err.Error())
		}
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case 'c':
			network.Send(net.EncodeChat("console", string(line[2:])))
		case 'h': // h = heartbeat
			network.Send(net.NewMsgHeartbeat())
		case 'p': // p = ping
			network.Send(net.NewMsgPing())
		}
	}
}
