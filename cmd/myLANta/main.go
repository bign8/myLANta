package main

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/bign8/myLANta"
)

func main() {
	exit := make(chan int, 10)
	log.Printf("Launching Server.")
	network := myLANta.RunServer(exit)

	go func() {
		hb := myLANta.Heartbeat{
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
			network.Outgoing <- &myLANta.Message{
				Target: myLANta.BroadcastTarget,
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
