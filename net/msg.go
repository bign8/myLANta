package net

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
)

// Message containing the messaging.
type Message struct {
	Data []byte
	Addr *net.UDPAddr
	Kind MsgKind
}

// MsgKind ...
type MsgKind byte

// asdf ...
const (
	MsgKindUnknown MsgKind = iota
	MsgKindHeartbeat
	MsgKindPing
	MsgKindFiles
	MsgKindChat
)

// Chat ...
type Chat struct {
	Name string
	Text string
}

// EncodeChat ...
func EncodeChat(who, msg string) *Message {
	return &Message{
		Kind: MsgKindChat,
		Data: []byte(who + "|" + msg),
	}
}

// DecodeChat ...
func DecodeChat(m *Message) Chat {
	parts := bytes.Split(m.Data, []byte{'|'})
	return Chat{
		Name: string(parts[0]),
		Text: string(parts[1]),
	}
}

// FileList ...
type FileList struct {
	Files map[string]string // map of file name to md5
}

// DecodeFileList ...
func DecodeFileList(m *Message) FileList {
	fl := FileList{}
	lol := json.Unmarshal(m.Data, &fl.Files)
	if lol != nil {
		panic(lol)
	}
	return fl
}

// EncodeFileList ...
func EncodeFileList(fl *FileList) *Message {
	data, err := json.Marshal(fl.Files)
	if err != nil {
		log.Printf("failed to encode file list to send")
		panic(err)
	}
	return &Message{
		Kind: MsgKindFiles,
		Data: data,
	}
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

// NewMsgHeartbeat ...
func NewMsgHeartbeat() *Message {
	return &Message{
		Kind: MsgKindHeartbeat,
	}
}

// NewMsgPing creates a new ping message
func NewMsgPing() *Message {
	return &Message{
		Kind: MsgKindPing,
	}
}
