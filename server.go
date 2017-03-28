package ddns

import (
	"log"
	"net"
	"fmt"
	"time"
	"strings"
	"io/ioutil"
	"github.com/miekg/dns"
	"github.com/stutiredboy/radix.v2/pool"
	"github.com/stutiredboy/radix.v2/redis"
)

// Server implements a DNS server.
type Server struct {
	c *dns.Client
	s *dns.Server
	p *pool.Pool
	/* current queries counter */
	n int64
	/* last queries counter for qps */
	l int64
}

// NewServer creates a new Server with the given options.
func NewServer(o Options) (*Server, error) {
	if err := o.validate(); err != nil {
		return nil, err
	}
	connect_timeout := time.Millisecond * time.Duration(o.ConnectTimeout)
	read_timeout := time.Millisecond * time.Duration(o.ReadTimeout)
	if o.Debug {
		log.Printf("create redis pool with connect_timeout: %s, read_timeout: %s", connect_timeout, read_timeout)
	}
	p, err := pool.NewCustom("tcp", o.Backend, o.PoolNum, connect_timeout, read_timeout, redis.DialTimeout)
	if err != nil {
		return nil, err
	}

	s := Server{
		c: &dns.Client{},
		s: &dns.Server{
			Net:  "udp",
			Addr: o.Bind,
		},
		p: p,
		n: 0,
		l: 0,
	}

	s.s.Handler = dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		// If no upstream proxy is present, drop the query:
		if len(o.Resolve) == 0 {
			dns.HandleFailed(w, r)
			return
		}

		if o.Debug {
			log.Printf("query %s from %s", r.Question[0].Name, w.RemoteAddr())
		}
		s.logq2b(r.Question[0].Name, w.RemoteAddr())
		// increase queries counter
		s.n += 1

		// Proxy Query:
		for _, addr := range o.Resolve {
			in, _, err := s.c.Exchange(r, addr)
			if err != nil {
				continue
			}
			w.WriteMsg(in)
			return
		}
		dns.HandleFailed(w, r)
	})
	return &s, nil
}

// ListenAndServe runs the server
func (s *Server) ListenAndServe() error {
	return s.s.ListenAndServe()
}

// Shutdown stops the server, closing its connection.
func (s *Server) Shutdown() error {
	return s.s.Shutdown()
}

func (s *Server) Dump(period int, saveto string) {
        qps := (s.n - s.l) / int64(period)
        log.Printf("total queries: %d, qps: %d", s.n, qps)
	if saveto != "" {
		err := ioutil.WriteFile(saveto, []byte(fmt.Sprintf("total queries: %d\n", s.n)), 644)
		if err != nil {
			log.Printf("dump statistics to %s err: %s", saveto, err)
		}
	}
        s.l = s.n
}

func (s *Server) logq2b(name string, addr net.Addr) error {
	name = strings.TrimSuffix(name, ".")
	clientip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return err
	}
	err = s.p.Cmd("SETEX", name, 120, clientip).Err
	if err != nil {
		log.Printf("setex %s as %s raise err: %s", name, clientip, err)
	}
	return err
}
