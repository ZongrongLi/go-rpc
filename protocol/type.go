/*
 * File: const.go
 * Project: protocol
 * File Created: Saturday, 6th April 2019 12:20:24 am
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Saturday, 6th April 2019 12:20:38 am
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright lizongrong - 2019
 */
package protocol

type SerializeType byte

const (
	SerializeTypeJson SerializeType = iota
	SerializeTypeOther
)

type MessageType byte

//请求类型
const (
	MessageTypeRequest MessageType = iota
	MessageTypeResponse
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
	RequestSeqKey     = "rpc_request_seq"
	RequestTimeoutKey = "rpc_request_timeout"
	MetaDataKey       = "rpc_meta_data"
)