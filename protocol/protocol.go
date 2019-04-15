/*
 * File: protocol.go
 * Project: protocol
 * File Created: Saturday, 6th April 2019 12:18:58 am
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Saturday, 6th April 2019 12:20:43 am
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null lizongrong - 2019
 */

////直接找了一个抄的
//-------------------------------------------------------------------------------------------------
//|2byte|1byte  |4byte       |4byte        | header length |(total length - header length - 4byte)|
//-------------------------------------------------------------------------------------------------
//|magic|version|total length|header length|     header    |                    body              |
//-------------------------------------------------------------------------------------------------
package protocol

import (
	"encoding/binary"
	"errors"
	"io"
)

type Header struct {
	Seq           uint64                 //序号, 用来唯一标识请求或响应
	MessageType   MessageType            //消息类型，用来标识一个消息是请求还是响应
	CompressType  CompressType           //压缩类型，用来标识一个消息的压缩方式
	SerializeType SerializeType          //序列化类型，用来标识消息体采用的编码方式
	StatusCode    StatusCode             //状态类型，用来标识一个请求是正常还是异常
	ServiceName   string                 //服务名
	MethodName    string                 //方法名
	Error         string                 //方法调用发生的异常
	MetaData      map[string]interface{} //其他元数据
}

type Message struct {
	*Header
	Data []byte
}

func NewMessage() *Message {
	msg := &Message{}
	msg.Header = &Header{}
	return msg
}

func (m Message) Clone() *Message {
	header := *m.Header
	c := new(Message)
	c.Header = &header
	c.Data = m.Data
	return c
}

type Protocol interface {
	NewMessage() *Message
	DecodeMessage(r io.Reader, s Serializer) (*Message, error)
	EncodeMessage(message *Message, s Serializer) []byte
}

var ProtocolMap = map[ProtocolType]Protocol{
	Default: RPCProtocol{},
}

type RPCProtocol struct {
}

func (RPCProtocol) NewMessage() *Message {
	return &Message{Header: &Header{}}
}

//DecodeMessage 从socket中读取数据并且包装成message
func (RPCProtocol) DecodeMessage(r io.Reader, s Serializer) (*Message, error) {
	first3bytes := make([]byte, 3)
	_, err := io.ReadFull(r, first3bytes)
	if err != nil {
		return nil, err
	}
	if !checkMagic(first3bytes[:2]) {
		err = errors.New("wrong protocol")
		return nil, err
	}
	totalLenBytes := make([]byte, 4)
	_, err = io.ReadFull(r, totalLenBytes)
	if err != nil {
		return nil, err
	}

	totalLen := int(binary.BigEndian.Uint32(totalLenBytes))
	if totalLen < 4 {
		err = errors.New("invalid total length")
		return nil, err
	}

	data := make([]byte, totalLen)
	_, err = io.ReadFull(r, data)
	headerLen := int(binary.BigEndian.Uint32(data[:4]))
	headerBytes := data[4 : headerLen+4]

	header := &Header{}
	err = s.Unmarshal(headerBytes, header)
	if err != nil {
		return nil, err
	}
	msg := Message{}
	msg.Header = header
	msg.Data = data[headerLen+4:]
	return &msg, nil
}

//EncodeMessage
func (RPCProtocol) EncodeMessage(msg *Message, s Serializer) []byte {
	first3bytes := []byte{0xab, 0xba, 0x00}
	headerBytes, _ := s.Marshal(msg.Header)

	totalLen := 4 + len(headerBytes) + len(msg.Data)
	totalLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(totalLenBytes, uint32(totalLen))

	data := make([]byte, totalLen+7)
	start := 0
	copyFullWithOffset(data, first3bytes, &start)
	copyFullWithOffset(data, totalLenBytes, &start)

	headerLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(headerLenBytes, uint32(len(headerBytes)))
	copyFullWithOffset(data, headerLenBytes, &start)
	copyFullWithOffset(data, headerBytes, &start)
	copyFullWithOffset(data, msg.Data, &start)
	return data
}

func checkMagic(bytes []byte) bool {
	return bytes[0] == 0xab && bytes[1] == 0xba
}

func copyFullWithOffset(dst []byte, src []byte, start *int) {
	copy(dst[*start:*start+len(src)], src)
	*start = *start + len(src)
}
