/*
 * File: gateway.go
 * Project: server
 * File Created: Sunday, 14th April 2019 10:44:27 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Monday, 15th April 2019 2:02:14 am
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright 2019 - 2019
 */
package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/tiancai110a/go-rpc/protocol"
	"github.com/tiancai110a/go-rpc/share/metadata"
)

const (
	HEADER_SEQ            = "rpc-header-seq"            //序号, 用来唯一标识请求或响应
	HEADER_MESSAGE_TYPE   = "rpc-header-message_type"   //消息类型，用来标识一个消息是请求还是响应
	HEADER_COMPRESS_TYPE  = "rpc-header-compress_type"  //压缩类型，用来标识一个消息的压缩方式
	HEADER_SERIALIZE_TYPE = "rpc-header-serialize_type" //序列化类型，用来标识消息体采用的编码方式
	HEADER_STATUS_CODE    = "rpc-header-status_code"    //状态类型，用来标识一个请求是正常还是异常
	HEADER_SERVICE_NAME   = "rpc-header-service_name"   //服务名
	HEADER_METHOD_NAME    = "rpc-header-method_name"    //方法名
	HEADER_ERROR          = "rpc-header-error"          //方法调用发生的异常
	HEADER_META_DATA      = "rpc-header-meta_data"      //其他元数据

)

func (s *SGServer) startGateway() {
	port := 5080
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	for err != nil && strings.Contains(err.Error(), "address already in use") {
		port++
		ln, err = net.Listen("tcp", ":"+strconv.Itoa(port))
	}
	if err != nil {
		log.Printf("error listening gateway: %s", err.Error())
	}

	log.Printf("gateway listenning on " + strconv.Itoa(port))
	go func() {
		err := http.Serve(ln, s)
		if err != nil {
			log.Printf("error serving http %s", err.Error())
		}
	}()
}

func (s *SGServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/invoke" {
		rw.WriteHeader(404)
		return
	}

	if r.Method != "POST" {
		rw.WriteHeader(405)
		return
	}

	request := protocol.NewMessage()
	request, err := parseHeader(request, r)
	if err != nil {
		rw.WriteHeader(400)
	}
	request, err = parseBody(request, r)
	if err != nil {
		rw.WriteHeader(400)
	}
	ctx := metadata.WithMeta(context.Background(), request.MetaData)
	response := request.Clone()
	response.MessageType = protocol.MessageTypeResponse
	response = s.process(ctx, request, response)
	s.writeHttpResponse(response, rw, r)
}

func parseBody(message *protocol.Message, request *http.Request) (*protocol.Message, error) {
	data, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	message.Data = data
	return message, nil
}

func parseHeader(message *protocol.Message, request *http.Request) (*protocol.Message, error) {
	headerSeq := request.Header.Get(HEADER_SEQ)
	seq, err := strconv.ParseUint(headerSeq, 10, 64)
	if err != nil {
		return nil, err
	}
	message.Seq = seq

	headerMsgType := request.Header.Get(HEADER_MESSAGE_TYPE)

	IMsgType, err := strconv.ParseUint(headerMsgType, 10, 64)
	if err != nil {
		return nil, err
	}
	msgType := (protocol.MessageType)(IMsgType)

	if err != nil {
		return nil, err
	}
	message.MessageType = msgType

	headerCompressType := request.Header.Get(HEADER_COMPRESS_TYPE)

	ICompressType, err := strconv.ParseUint(headerCompressType, 10, 64)
	if err != nil {
		return nil, err
	}
	compressType := (protocol.CompressType)(ICompressType)
	if err != nil {
		return nil, err
	}
	message.CompressType = compressType

	headerSerializeType := request.Header.Get(HEADER_SERIALIZE_TYPE)
	ISerializeType, err := strconv.ParseUint(headerSerializeType, 10, 64)
	if err != nil {
		return nil, err
	}
	serializeType := (protocol.SerializeType)(ISerializeType)
	if err != nil {
		return nil, err
	}
	message.SerializeType = serializeType

	headerStatusCode := request.Header.Get(HEADER_STATUS_CODE)
	IStatusCode, err := strconv.ParseUint(headerStatusCode, 10, 64)
	statusCode := (protocol.StatusCode)(IStatusCode)

	if err != nil {
		return nil, err
	}
	message.StatusCode = statusCode

	serviceName := request.Header.Get(HEADER_SERVICE_NAME)
	message.ServiceName = serviceName

	methodName := request.Header.Get(HEADER_METHOD_NAME)
	message.MethodName = methodName

	errorMsg := request.Header.Get(HEADER_ERROR)
	message.Error = errorMsg

	headerMeta := request.Header.Get(HEADER_META_DATA)
	meta := make(map[string]interface{})
	err = json.Unmarshal([]byte(headerMeta), &meta)
	if err != nil {
		return nil, err
	}
	message.MetaData = meta

	return message, nil
}

func (s *SGServer) writeHttpResponse(message *protocol.Message, rw http.ResponseWriter, r *http.Request) {
	header := rw.Header()
	header.Set(HEADER_SEQ, string(message.Seq))
	header.Set(HEADER_MESSAGE_TYPE, strconv.FormatUint((uint64)(message.MessageType), 10))
	header.Set(HEADER_COMPRESS_TYPE, strconv.FormatUint((uint64)(message.CompressType), 10))
	header.Set(HEADER_SERIALIZE_TYPE, strconv.FormatUint((uint64)(message.SerializeType), 10))
	header.Set(HEADER_STATUS_CODE, strconv.FormatUint((uint64)(message.StatusCode), 10))
	header.Set(HEADER_SERVICE_NAME, message.ServiceName)
	header.Set(HEADER_METHOD_NAME, message.MethodName)
	header.Set(HEADER_ERROR, message.Error)
	metaDataJson, _ := json.Marshal(message.MetaData)
	header.Set(HEADER_META_DATA, string(metaDataJson))

	_, _ = rw.Write(message.Data)
}
