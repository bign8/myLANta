package model

// State gives the list of files.
type State interface {

	// Conns gives the list of client connections (not sure if we need this).
	Conns() []string

	// Files gives the list of actively hosted files.
	Files() map[string]string // Filename to MD5

	// Fetch a particular file based on MD5 hash.
	Fetch(name string) (byte, error)

	// Serve adds a particular file listing to the list.
	Serve(name string, bits []byte) error

	// Title sets the name of a person (returns value if name parameter is empty).
	Title(name string) string
}
