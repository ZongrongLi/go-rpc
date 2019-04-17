/*
 * File: http_interceptor.go
 * Project: server
 * File Created: Wednesday, 17th April 2019 3:44:30 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Wednesday, 17th April 2019 3:44:44 pm
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * Copyright nil - 2019
 */
package server

type HttpInterceptor struct {
	DefaultServerWrapper
}

//根据请求方法名等信息生成链路信息
//通过rpc metadata传递追踪信息
func (*HttpInterceptor) WrapHttpHandleRequest(s *SGServer, requestFunc HandleRequestFunc) HandleRequestFunc {
	return requestFunc
}
func (*HttpInterceptor) WrapHttpHandleResponse(s *SGServer, serveFunc ServeFunc) ServeFunc {
	return serveFunc
}
