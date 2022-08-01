package conf_test

import (
	"context"
	"fmt"
	"net"
	"testing"
)

func TestGetIPAddr(t *testing.T) {
	resolver := net.Resolver{}
	ips, err := resolver.LookupIPAddr(context.Background(), "ecloud.10086.cn")
	if err != nil {
		panic(err)
	}
	for _, ip := range ips {
		fmt.Println(ip, ip.String(), ip.IP)
	}
	fmt.Println("----------")
	// 可以判断一个网站是否支持ipv6
	ipss, err := resolver.LookupIP(context.Background(), "ip6", "ecloud.10086.cn")
	if err != nil {
		panic(err)
	}
	for _, ip := range ipss {
		fmt.Println(ip, ip.String(), ip.To4(), ip.To16())
	}
}
