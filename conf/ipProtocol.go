package conf

type IPProtocol string

var (
	IPV4 = IPProtocol("ip4")
	IPV6 = IPProtocol("ip6")
)

var IPProtocol2Gauge = map[IPProtocol]float64{
	IPV4: 4,
	IPV6: 6,
}
