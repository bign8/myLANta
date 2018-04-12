package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bign8/myLANta/mylanta"
	"github.com/bign8/myLANta/web"
)

var portz = flag.String("PORT", "8080", "port to serve on")

func main() {
	exit := make(chan int, 10)
	log.Printf("Launching Server.")
	network := mylanta.RunServer(exit)

	go func() {
		hb := mylanta.Heartbeat{
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
			time.Sleep(time.Second)
			network.Outgoing <- &mylanta.Message{
				Target: mylanta.BroadcastTarget,
				Raw:    append(bytes, msg...),
			}
		}
	}()

	flag.Parse()
	log.Println("Serving on :" + *portz)
	go func() {
		panic(http.ListenAndServe(":"+*portz, web.New(network)))
	}()

	buf := make([]byte, 1)
	os.Stdin.Read(buf)
	close(exit)
	log.Printf("goodbye")
}
