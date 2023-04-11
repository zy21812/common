package tools

import (
	"crypto/md5"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

func GetIpList() []string {
	ips := []string{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logrus.Error("getIp ", err)
	} else {
		for _, address := range addrs {
			// 检查ip地址判断是否回环地址
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ips = append(ips, ipnet.IP.String())
				}
			}
		}
	}
	ips = append(ips, "127.0.0.1")
	return ips
}

func Md5(data string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}
