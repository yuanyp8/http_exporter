package conf

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"hash/fnv"
	"net"
	"time"
)

type IPProtocol string

var (
	IPV4 = IPProtocol("ip4")
	IPV6 = IPProtocol("ip6")
)

var IPProtocol2Gauge = map[IPProtocol]float64{
	IPV4: 4,
	IPV6: 6,
}

// ChooseProtocol 确定给定的target域名/ip对应的ip protocol
func (h *HTTPProbe) ChooseProtocol(ctx context.Context, target string) (ip *net.IPAddr, lookupTime float64, err error) {

	// registry.MustRegister(probeDNSLookupTimeSeconds, probeIPProtocolGauge, probeIPAddrHash)
	protoStr := string(h.IPProtocol)
	var fallbackProtocol IPProtocol
	// 默认是IPV6，如失败则回滚到ipv4
	switch h.IPProtocol {
	case IPV6:
		fallbackProtocol = IPV4
	case IPV4:
		fallbackProtocol = IPV6
	default:
		h.IPProtocol = IPV6
		fallbackProtocol = IPV4
	}

	l.Info("Resolving target address", zap.String("target", target), zap.String("ip_protocol", protoStr))

	// 记录dns解析时间
	resolveStart := time.Now()
	defer func() {
		lookupTime = time.Since(resolveStart).Seconds()
		probeDNSLookupTimeSeconds.Add(lookupTime)
	}()

	// 开始 dns 解析
	resolver := &net.Resolver{}

	// 如果不允许协议降级，根据指定的协议进行处理，失败则返回
	if !h.IPProtocolFallback {
		// 基于给定的协议（ip/ip4/ip6）给出host对应的ip列表
		ips, err := resolver.LookupIP(ctx, protoStr, target)
		if err == nil {
			for _, ip := range ips {
				// 解析成功了,只要匹配到第一个ip就返回
				l.Info("Resolved target address", zap.String("target", target), zap.String("ip", ip.String()))
				probeIPProtocolGauge.Set(IPProtocol2Gauge[h.IPProtocol])
				probeIPAddrHash.Set(ipHash(ip))
				return &net.IPAddr{IP: ip}, lookupTime, nil
			}
		}
		// 根据本地的dns解析这个target失败了
		l.Error("Resolution with IP protocol failed", zap.String("target", target), zap.String("ip_protocol", protoStr), zap.Error(err))
		return nil, 0.0, err
	}

	// 允许协议降级,如果指定的ip protocol失败了，则转换协议测试
	// LookupIPAddr()支持ip4和ip6
	ips, err := resolver.LookupIPAddr(ctx, target)
	if err != nil {
		l.Error("Resolution with IP protocol failed", zap.String("target", target), zap.Error(err))
		return nil, 0.0, err
	}

	// Return the IP in the requested protocol.
	var fallback *net.IPAddr
	for _, ip := range ips {
		// 现在的ips列表不确定是ip4还是ip6解析成功了
		switch h.IPProtocol {
		case IPV4:
			// To4()结果非空则证明为ip4地址
			if ip.IP.To4() != nil {
				l.Info("Resolved target address", zap.String("target", target), zap.String("ip", ip.String()))
				probeIPProtocolGauge.Set(IPProtocol2Gauge[IPV4])
				probeIPAddrHash.Set(ipHash(ip.IP))
				return &ip, lookupTime, nil
			}
			// ip6 as fallback,此ip应用ipv6协议去解析
			fallback = &ip

		case IPV6:
			// 非4即6
			if ip.IP.To4() == nil {
				l.Info("Resolved target address", zap.String("target", target), zap.String("ip", ip.String()))
				probeIPProtocolGauge.Set(IPProtocol2Gauge[IPV6])
				probeIPAddrHash.Set(ipHash(ip.IP))
				return &ip, lookupTime, nil
			}
			// ip4 as fallback
			fallback = &ip
		}
	}
	// Unable to find ip and no fallback set.
	if fallback == nil || !h.IPProtocolFallback {
		return nil, 0.0, fmt.Errorf("unable to find ip; no fallback")
	}
	if fallbackProtocol == IPV4 {
		probeIPProtocolGauge.Set(IPProtocol2Gauge[IPV4])
	} else {
		probeIPProtocolGauge.Set(IPProtocol2Gauge[IPV6])
	}
	probeIPAddrHash.Set(ipHash(fallback.IP))
	l.Info("Resolved target address", zap.String("target", target), zap.String("ip", fallback.String()))
	return fallback, lookupTime, nil
}

// 将IP地址进行Hash
func ipHash(ip net.IP) float64 {
	h := fnv.New32a()
	if ip.To4() != nil {
		h.Write(ip.To4())
	} else {
		h.Write(ip.To16())
	}
	return float64(h.Sum32())
}
