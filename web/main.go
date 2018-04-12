package main

import "net/http"

var files = map[string][]byte{}

func main() {
	println("Serving on ")

	// listen to UDP requests and respond as necessary

	panic(http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("who needs internet anyway\n"))
	})))
}

func handle(msg string) {
	switch msg[0] {
	case '?': // who is there and what files are you serving
	case 'p': // list of peers
	case 'f': // list of files
	}
}
