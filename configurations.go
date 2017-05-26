package ddns

import (
	"net"
	"errors"
	"strings"
)

// Configurations for server
type Configurations struct {
    // host:port
    Listen string
    // [host1:port1, host2:port2]
    NameServers []string
    // Seconds, dump stats to StatsFile periodically
    StatsPeriod int
    StatsFile string
    Backends map[int]string
    // backend connection pool numbers
    PoolNum int
    // channel numbers, must little than PoolNum
    ChanNum int
    // millseconds
    ConnectTimeout int
    // millseconds
    ReadTimeout int
    Debug bool
}

func (c *Configurations) validate() error {
	if !strings.Contains(c.Listen, ":") {
		c.Listen += ":53"
	}
	if l := len(c.Listen); l >= 4 && c.Listen[l-4:] == ":dns" {
		c.Listen = c.Listen[:l-4] + ":53"
	}
	if c.Listen[0] == ':' {
		c.Listen= "0.0.0.0" + c.Listen
	}

	for i, res := range c.NameServers {
		if !strings.Contains(res, ":") {
			res += ":53"
		}
		addr, err := net.ResolveUDPAddr("udp", res)
		if err != nil {
			return err
		}
		c.NameServers[i] = addr.String()
	}
	/* ensure have full coverage index */
	for i := 0 ; i < len(c.Backends) ; i++ {
		_, ok := c.Backends[i]
		if ok == false {
			err := errors.New("Wrong backends hash set, hash should full converage [0, len_of_Backends)")
			return err
		}
	}
	return nil
}
