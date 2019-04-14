package protocol

import (
	"strconv"
	"testing"
)

// type MessageType byte

// //请求类型
// const (
// 	MessageTypeRequest MessageType = iota
// 	MessageTypeResponse
// 	MessageTypeHeartbeat
// )

func TestIota(t *testing.T) {
	s := "1"
	a, _ := strconv.ParseUint(s, 10, 64)
	i := (MessageType)(a)

	switch i {
	case MessageTypeHeartbeat:
		t.Log("MessageTypeHeartbeat")
	case MessageTypeRequest:
		t.Log("MessageTypeRequest")
	case MessageTypeResponse:
		t.Log("MessageTypeResponse")
	}

	return
}
