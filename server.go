package ddns

import (
	"log"
	"time"
	"github.com/miekg/dns"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
)

// global vairable
var RedisConnTimeout = time.Duration(0)

// Server implements a DNS server.
type Server struct {
	c *dns.Client
	s *dns.Server
	p *pool.Pool
}

// Custom redis.Client Dial, add timeout
func DDNSDial(network, addr string) (*redis.Client, error) {
	return redis.DialTimeout(network, addr, RedisConnTimeout)
}

// NewServer creates a new Server with the given options.
func NewServer(o Options) (*Server, error) {
	if err := o.validate(); err != nil {
		return nil, err
	}
	RedisConnTimeout = time.Millisecond * time.Duration(o.Timeout)
	if o.Debug {
		log.Printf("create redis pool with timeout: %d", RedisConnTimeout)
	}
	pool, err := pool.NewCustom("tcp", o.Backend, o.PoolNum, DDNSDial)
	if err != nil {
		return nil, err
	}

	s := Server{
		c: &dns.Client{},
		s: &dns.Server{
			Net:  "udp",
			Addr: o.Bind,
		},
		p: pool,
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
		s.logq2b(r.Question[0].Name, w.RemoteAddr(), s.p)

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
