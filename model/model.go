package model

import "errors"

// Various errors
var (
	ErrFileDNE = errors.New("myLANta: file does not exist")
)

// FileService gives the list of files.
type FileService interface {
	// Files gives the list of actively hosted files (sorted alphabetically).
	Files() map[string]string // hash -> name

	// Fetch a particular file based on MD5 hash.
	Fetch(hash string) ([]byte, error)

	// Serve adds a particular file listing to the list.
	Serve(name string, bits []byte) error
}

// ChatService gives programatic chat access.
type ChatService interface {
	// Peers gives the list of peers within the network.
	Peers() map[string]string // addr -> name

	// Title gives the name of a person (assigns a new name iff not empty).
	Title(name string) string

	// Message broadcasts a message.
	Message(who, what string) error

	// Listen gives active backlog of messages and a stream of new ones.
	Listen() ([]*Message, <-chan *Message, func())
}

// MyLANta is named here to make it painfully obvious what the bigger picture is.
type MyLANta interface {
	FileService
	ChatService
}

// Message is a type of payload that is transmitted.
type Message struct {
	Addr string // empty means broadcast to everybody
	Data []byte
}
