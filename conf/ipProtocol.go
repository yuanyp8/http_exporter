package conf

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"hash/fnv"
	"net"
	"time"
)

type IPProtocol string

var (
	IPV4 = IPProtocol("ipv4")
	IPV6 = IPProtocol("ipv6")
)

var IPProtocol2Gauge = map[IPProtocol]float64{
	IPV4: 4,
	IPV6: 6,
}

func (h *HTTPProbe) chooseProtocol(ctx context.Context, target string, registry *prometheus.Registry) (ip *net.IPAddr, lookupTime float64, err error) {
	probeDNSLookupTimeSeconds := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_dns_lookup_time_seconds",
		Help: "Returns the time taken for probe dns lookup in seconds",
	})

	probeIPProtocolGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_ip_protocol",
		Help: "Specifies whether probe ip protocol is IP4 or IP6",
	})

	probeIPAddrHash := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_ip_addr_hash",
		Help: "Specifies the hash of IP address. It's useful to detect if the IP address changes.",
	})
	registry.MustRegister(probeDNSLookupTimeSeconds, probeIPProtocolGauge, probeIPAddrHash)
	protoStr := string(h.IPProtocol)
	var fallbackProtocol IPProtocol
	switch h.IPProtocol {
	case IPV6:
		fallbackProtocol = IPV4
	case IPV4:
		fallbackProtocol = IPV6
	default:
		h.IPProtocol = IPV6
		fallbackProtocol = IPV4
	}

	l.Info("Resolving target address",
		zap.String("target", target),
		zap.String("ip_protocol", protoStr),
	)

	// 记录dns解析时间
	resolveStart := time.Now()
	defer func() {
		lookupTime = time.Since(resolveStart).Seconds()
		probeDNSLookupTimeSeconds.Add(lookupTime)
	}()

	resolver := &net.Resolver{}

	// 不允许协议降级
	if !h.IPProtocolFallback {
		ips, err := resolver.LookupIP(ctx, string(h.IPProtocol), target)
		if err == nil {
			for _, ip := range ips {
				l.Info("Resolved target address", zap.String("target", target), zap.String("ip", ip.String()))
				probeIPProtocolGauge.Set(IPProtocol2Gauge[h.IPProtocol])
				probeIPAddrHash.Set(ipHash(ip))
				return &net.IPAddr{IP: ip}, lookupTime, nil
			}
		}
		l.Error("Resolution with IP protocol failed", zap.String("target", target), zap.String("ip_protocol", protoStr), zap.Error(err))
		return nil, 0.0, err
	}

	// 允许协议降级
	ips, err := resolver.LookupIPAddr(ctx, target)
	if err != nil {
		l.Error("Resolution with IP protocol failed", zap.String("target", target), zap.Error(err))
		return nil, 0.0, err
	}

	// Return the IP in the requested protocol.
	var fallback *net.IPAddr
	for _, ip := range ips {
		switch h.IPProtocol {
		case IPV4:
			if ip.IP.To4() != nil {
				l.Info("Resolved target address", zap.String("target", target), zap.String("ip", ip.String()))
				probeIPProtocolGauge.Set(IPProtocol2Gauge[IPV4])
				probeIPAddrHash.Set(ipHash(ip.IP))
				return &ip, lookupTime, nil
			}
			// ip4 as fallback
			fallback = &ip

		case IPV6:
			if ip.IP.To4() == nil {
				l.Info("Resolved target address", zap.String("target", target), zap.String("ip", ip.String()))
				probeIPProtocolGauge.Set(IPProtocol2Gauge[IPV6])
				probeIPAddrHash.Set(ipHash(ip.IP))
				return &ip, lookupTime, nil
			}
			// ip6 as fallback
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
