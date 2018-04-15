package model

// Service gives the list of files.
type Service interface {

	// Peers gives the list of peers within the network.
	Peers() map[string]string // hash -> name

	// Files gives the list of actively hosted files (sorted alphabetically).
	Files() map[string]string // hash -> name

	// Fetch a particular file based on MD5 hash.
	Fetch(hash string) ([]byte, error)

	// Serve adds a particular file listing to the list.
	Serve(name string, bits []byte) error

	// Title gives the name of a person (assigns a new name iff not empty).
	Title(name string) string
}
