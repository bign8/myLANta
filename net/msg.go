package net

import "encoding/binary"

// Message containing the messaging.
type Message struct {
	Raw    []byte
	Target int16
	Kind   MsgKind
}

type MsgKind byte

const (
	MsgKindUnknown MsgKind = iota
	MsgKindHeartbeat
	MsgKindPing
	MsgKindFiles
	MsgKindChat
)

type Chat struct {
	Name string
	Text string
}

func decodeChat(m *Message) Chat {
	length := binary.LittleEndian.Uint16(m.Raw[1:3])
	if length > 1500 {
		panic("TOO BIG MSG")
	}
	return Chat{
		Text: string(m.Raw[3 : 3+length]),
	}
}

type FileList struct {
	Files map[string]string // map of file name to md5
}

// Heartbeat information.
type Heartbeat struct {
	Name string
}

func decodeHeartbeat(m *Message) Heartbeat {
	hb := Heartbeat{}
	// lol := json.Unmarshal(m.Raw[2:], &hb)
	// if lol != nil {
	// 	panic(lol)
	// }
	return hb
}
