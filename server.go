package ddns

import (
	"log"
	"github.com/miekg/dns"
)

// Server implements a DNS server.
type Server struct {
	c *dns.Client
	s *dns.Server
}

// NewServer creates a new Server with the given options.
func NewServer(o Options) (*Server, error) {
	if err := o.validate(); err != nil {
		return nil, err
	}

	s := Server{
		c: &dns.Client{},
		s: &dns.Server{
			Net:  o.Net,
			Addr: o.Bind,
		},
	}

	s.s.Handler = dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		// If no upstream proxy is present, drop the query:
		if len(o.Resolve) == 0 {
			dns.HandleFailed(w, r)
			return
		}

		// query *domain*
		log.Printf("%s", r.Question[0].Name)
		// query *client*
		log.Printf("%s", w.RemoteAddr())

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
