package dnsp

import (
	"fmt"
	"net"
	"strings"
)

// Options can be passed to NewServer().
type Options struct {
	Net     string
	Bind    string
	Resolve []string
}

// validate verifies that the options are correct.
func (o *Options) validate() error {
	if o.Net == "" {
		o.Net = "udp"
	}
	if o.Net != "udp" && o.Net != "tcp" {
		return fmt.Errorf("net: must be one of 'tcp', 'udp'")
	}

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

	return nil
}
