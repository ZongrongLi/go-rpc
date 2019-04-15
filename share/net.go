/*
 * File: net.go
 * Project: share
 * File Created: Thursday, 11th April 2019 6:23:26 pm
 * Author: lizongrong (389006500@qq.com)
 * -----
 * Last Modified: Friday, 12th April 2019 1:21:30 am
 * Modified By: lizongrong (389006500@qq.com>)
 * -----
 * null 2019 - 2019
 */
package share

import (
	"net"

	"github.com/golang/glog"
)

var localIpV4 string

func init() {
	addrs, err := net.InterfaceAddrs()
	glog.Info("addr:", addrs)
	if err != nil {
		glog.Fatal("check net interfaces error:", err.Error())
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localIpV4 = ipnet.IP.String()
				break
			}
		}
	}
}

func LocalIpV4() string {
	return localIpV4
}
