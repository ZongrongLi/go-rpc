/*
 * File: gateway.go
 * Project: server
 * File Created: Sunday, 14th April 2019 10:44:27 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Monday, 15th April 2019 2:02:14 am
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */
package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/tiancai110a/go-rpc/protocol"
	Service "github.com/tiancai110a/go-rpc/service"
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

func (s *SGServer) startHttpsGateway(port int, cert, key string) {
	beginPoint := &Middleware{
		F:    DefaultHTTPServeFunc,
		GoOn: nil,
	}
	s.option.HttpBeginPoint = chain(beginPoint, s.option.HttpWraper...)

	go func() {
		addr := ":" + strconv.Itoa(port)
		glog.Infof("Start to listening the incoming requests on https address: %s", addr)
		glog.Info(http.ListenAndServeTLS(addr, s.option.Sslcert, s.option.Sslkey, s).Error())
		glog.Info("htttps listenning on " + strconv.Itoa(port))
	}()
}

func (s *SGServer) startGateway(port int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	for err != nil && strings.Contains(err.Error(), "address already in use") {
		port++
		ln, err = net.Listen("tcp", ":"+strconv.Itoa(port))
	}
	if err != nil {
		glog.Error("error listening gateway: %s", err.Error())
	}

	glog.Info("gateway listenning on " + strconv.Itoa(port))

	go func() {
		err := http.Serve(ln, s)
		if err != nil {
			glog.Error("error serving http %s", err.Error())
		} else {
			glog.Info("Http server at port: ", port)
		}
	}()
}

func (s *SGServer) Group(t Service.MethodType, path string) *Service.MapRouterFunc {

	path = strings.Trim(path, "/")
	pathstr := strings.Split(path, "/")

	fm := &Service.PostGroup2Func
	if t == Service.GET {
		fm = &Service.GetGroup2Func
	}

	cc := ""
	gname := ""
	for i := 0; i < len(pathstr); i++ {
		gname = gname + cc + pathstr[i]
		cc = "_"
	}

	if _, ok := (*fm)[gname]; !ok {
		m := Service.MapRouterFunc{}
		(*fm)[gname] = &m
	}

	return (*fm)[gname]
}

func parsePath(path string) (gname, mname string, err error) {
	//解析路径
	mname = ""
	gname = ""
	path = strings.Trim(path, "/")
	pathstr := strings.Split(path, "/")
	cc := ""

	for i := 0; i < len(pathstr)-1; i++ {
		gname = gname + cc + pathstr[i]
		cc = "_"
	}
	if len(path) > 0 {
		mname = pathstr[len(pathstr)-1]
	}

	mname = strings.Trim(mname, "/")
	gname = strings.Trim(gname, "/")
	return
}

func (s *SGServer) Use(f HTTPServeFunc) {
	s.option.HttpWraper = append(s.option.HttpWraper, f)
}
func (s *SGServer) UseGroup(groupname string, f HTTPServeFunc) {
	groupname = strings.Trim(groupname, "/")
	mw, ok := s.option.HttpGroupBeginPoint[groupname]
	if !ok {
		beginPoint := &Middleware{
			F:    DefaultHTTPServeFunc,
			GoOn: nil,
		}
		mw = chain(beginPoint, f)
	} else {
		mw = chain(mw, f)
	}
	s.option.HttpGroupBeginPoint[groupname] = mw
}

func sendResponse(rw http.ResponseWriter, rsp *Service.Resp) {
	buff, err := json.Marshal(rsp)
	if err != nil {
		rw.WriteHeader(500)
		return
	}

	_, _ = rw.Write(buff)
	return
}

func (s *SGServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	//调用中间件
	s.option.HttpBeginPoint.Next(&rw, r)

	request := protocol.NewMessage()
	//var err error
	//request, err = parseHeader(request, r)
	// if err != nil {
	// 	rw.WriteHeader(400)
	// }

	ctx := context.Background()
	if request.MetaData != nil {
		ctx = metadata.WithMeta(context.Background(), request.MetaData)
	}

	gname, mname, err := parsePath(r.URL.Path)

	if err != nil {
		rw.WriteHeader(500)
		return
	}

	//调用群组中间件
	beginpoint, ojbk := s.option.HttpGroupBeginPoint[gname]

	if ojbk {
		beginpoint.Next(&rw, r)
	}

	if rw.Header().Get("Statuscode") == "0" || rw.Header().Get("Statuscode") == "" {
	} else {
		rsp := Service.NewResp()
		statuscode, err := strconv.ParseInt(rw.Header().Get("Statuscode"), 10, 64)
		if err != nil {
			rw.WriteHeader(500)
			glog.Error("strconv failed err:", err)
			return
		}
		rsp.Statuscode = int(statuscode)
		rsp.Message = rw.Header().Get("Message")
		sendResponse(rw, &rsp)
		return
	}

	ctx = context.WithValue(ctx, Service.Groupname, gname)
	ctx = context.WithValue(ctx, Service.Methodpath, mname)

	response := request.Clone()
	response.MessageType = protocol.MessageTypeResponse

	request.ServiceName = "RouterService"
	if r.Method == "POST" {
		request.MethodName = "PostRouter"
	} else if r.Method == "GET" {
		request.MethodName = "GetRouter"
	} else {
		rw.WriteHeader(500)
		return
	}

	//解析body中的参数放到ctx 里面
	ctx, err = parseBody(ctx, r)
	if err != nil {
		glog.Error("parse params failed err: ", err)
	}

	response = s.process(ctx, request, response)

	rsp := Service.NewRouterResponse()
	err = s.serializer.Unmarshal(response.Data, &rsp)
	if err != nil {
		glog.Error("http response unmarshal failed err: ", err)
		rw.WriteHeader(500)
		return
	}
	sendResponse(rw, rsp.Data)
	//s.writeHttpResponse(response, rw, r)
}

func parseBody(ctx context.Context, request *http.Request) (context.Context, error) {
	if err := request.ParseForm(); err != nil {
		return ctx, errors.New("wrong params")
	}
	for k, v := range request.Form {
		glog.Info("params", k, "    :", v[0])
		ctx = context.WithValue(ctx, k, v[0])
	}
	return ctx, nil
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
