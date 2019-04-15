/*
 * File: const.go
 * Project: protocol
 * File Created: Saturday, 6th April 2019 12:20:24 am
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Saturday, 6th April 2019 12:20:38 am
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null lizongrong - 2019
 */
package protocol

type SerializeType byte

const (
	SerializeTypeJson SerializeType = iota
	SerializeTypeMsgpack
	SerializeTypeOther
)

type MessageType byte

//请求类型
const (
	MessageTypeRequest MessageType = iota
	MessageTypeResponse
	MessageTypeHeartbeat
)

type CompressType byte

const (
	CompressTypeNone CompressType = iota
)

type StatusCode byte

const (
	StatusOK StatusCode = iota
	StatusError
)

type ProtocolType byte

const (
	Default ProtocolType = iota
)

const (
	RequestSeqKey = "rpc_request_seq"
	AuthKey       = "rpc_auth"
	MetaDataKey   = "rpc_meta_data"
)
