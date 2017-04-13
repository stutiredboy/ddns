package ddns

import (
	"log"
	"log/syslog"
	"net"
	"fmt"
	"time"
	"strings"
	"io/ioutil"
	"github.com/miekg/dns"
	"github.com/stutiredboy/radix.v2/pool"
	"github.com/stutiredboy/radix.v2/redis"
)

type qinfo struct {
	name string
	addr net.Addr
}

// Server implements a DNS server.
type Server struct {
	c *dns.Client
	s *dns.Server
	p *pool.Pool
	/* current queries counter */
	n int64
	/* last queries counter for qps */
	l int64
	/* log failed counter */
	f int64
	sysLog *syslog.Writer
	log_chan map[int]chan qinfo
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
	/* log_chan initial begin */
	log_chan := make(map[int]chan qinfo)

	for i := 0; i < o.ChanNum ; i++ {
		log_chan[i] = make(chan qinfo, 10)
	}
	/* log_chan initial finish */

	sysLog, err := syslog.Dial("unixgram", "/dev/log", syslog.LOG_DEBUG|syslog.LOG_LOCAL5, "ddns")
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
		f: 0,
		sysLog: sysLog,
		//log_chan: make(chan qinfo, 5),
		log_chan: log_chan,
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
		// send query info to channel
		chan_index := int(hash(r.Question[0].Name)) % o.ChanNum
		select {
			case s.log_chan[chan_index] <- qinfo{r.Question[0].Name, w.RemoteAddr()}:
			default:
				s.f += 1
				log.Printf("receive query %s %s, but channel %d full", r.Question[0].Name, w.RemoteAddr(), chan_index)
		}
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
        log.Printf("total queries: %d, qps: %d, log failed: %d", s.n, qps, s.f)
	if saveto != "" {
		err := ioutil.WriteFile(saveto, []byte(fmt.Sprintf("total queries: %d\n", s.n)), 644)
		if err != nil {
			log.Printf("dump statistics to %s err: %s", saveto, err)
		}
	}
        s.l = s.n
}

func (s *Server) log2b(name string, addr net.Addr) error {
	name = strings.TrimSuffix(name, ".")
	clientip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return err
	}
	s.sysLog.Debug(fmt.Sprintf("query %s from %s", name, clientip))
	err = s.p.Cmd("SETEX", name, 120, clientip).Err
	return err
}

func (s *Server) Log2b(chan_index int) {
	log.Printf("listening to channel %d" , chan_index)
	for {
		query := <- s.log_chan[chan_index]
		err := s.log2b(query.name, query.addr)
		if err != nil {
			log.Printf("channel %d log2b %s %s raise err: %s", chan_index, query.name, query.addr, err)
		}
	}
}
