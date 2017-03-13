package ddns

import (
	"net"
	"strings"
)

// Options can be passed to NewServer().
type Options struct {
	Bind    string
	Resolve []string
	Backend string
	PoolNum int
	Timeout int
	Debug bool
}

// validate verifies that the options are correct.
func (o *Options) validate() error {
	if !strings.Contains(o.Bind, ":") {
		o.Bind += ":53"
	}
	if l := len(o.Bind); l >= 4 && o.Bind[l-4:] == ":dns" {
		o.Bind = o.Bind[:l-4] + ":53"
	}
	if o.Bind[0] == ':' {
		o.Bind = "0.0.0.0" + o.Bind
	}

	for i, res := range o.Resolve {
		if !strings.Contains(res, ":") {
			res += ":53"
		}
		addr, err := net.ResolveUDPAddr("udp", res)
		if err != nil {
			return err
		}
		o.Resolve[i] = addr.String()
	}

	if o.Backend == "" {
		o.Backend = "127.0.0.1:6379"
	}

	if o.PoolNum == 0 {
		o.PoolNum = 10
	}

	if o.Timeout == 0 {
		o.Timeout = 100
	}

	return nil
}
