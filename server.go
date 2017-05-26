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
	pools map[int]*pool.Pool
	/* current queries counter */
	n int64
	/* last queries counter for qps */
	l int64
	/* log failed counter */
	f int64
	sysLog *syslog.Writer
	logChan map[int]map[int]chan qinfo
	lenBackends int
}

// NewServer creates a new Server with the given options.
func NewServer(c Configurations) (*Server, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	connectTimeout := time.Millisecond * time.Duration(c.ConnectTimeout)
	readTimeout := time.Millisecond * time.Duration(c.ReadTimeout)
	if c.Debug {
		log.Printf("create redis pool with connectTimeout: %s, readTimeout: %s", connectTimeout, readTimeout)
	}
	pools := make(map[int]*pool.Pool)
	logChan := make(map[int]map[int]chan qinfo)
	for index, backend := range c.Backends {
		p, err := pool.NewCustom("tcp", backend, c.PoolNum, connectTimeout, readTimeout, redis.DialTimeout)
		if err != nil {
			return nil, err
		}
		pools[index] = p
		_logChan := make(map[int]chan qinfo)
		for i := 0; i < c.ChanNum ; i++ {
			_logChan[i] = make(chan qinfo, 10)
		}
		logChan[index] = _logChan
	}

	sysLog, err := syslog.Dial("unixgram", "/dev/log", syslog.LOG_DEBUG|syslog.LOG_LOCAL5, "ddns")
	if err != nil {
		return nil, err
	}

	s := Server{
		c: &dns.Client{},
		s: &dns.Server{
			Net:  "udp",
			Addr: c.Listen,
		},
		pools: pools,
		n: 0,
		l: 0,
		f: 0,
		sysLog: sysLog,
		logChan: logChan,
		lenBackends: len(c.Backends),
	}

	s.s.Handler = dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		// If no upstream proxy is present, drop the query:
		if len(c.NameServers) == 0 {
			dns.HandleFailed(w, r)
			return
		}

		if c.Debug {
			log.Printf("query %s from %s", r.Question[0].Name, w.RemoteAddr())
		}
		// send query info to channel
		name := strings.ToLower(strings.TrimSuffix(r.Question[0].Name, "."))
		// Backend and Channel must use different hash method
		backendIndex := backendHash(name) % s.lenBackends
		chanIndex := channelHash(name) % c.ChanNum
		select {
			case s.logChan[backendIndex][chanIndex] <- qinfo{name, w.RemoteAddr()}:
			default:
				s.f++
				log.Printf("receive query %s %s, but backend %d channel%d full", r.Question[0].Name, w.RemoteAddr(), backendIndex, chanIndex)
		}
		// increase queries counter
		s.n++

		// Proxy Query:
		for _, addr := range c.NameServers {
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

// Dump the stats of ddns
func (s *Server) Dump(period int, saveto string) {
        qps := (s.n - s.l) / int64(period)
        log.Printf("total queries: %d, qps: %d, log failed: %d", s.n, qps, s.f)
	if saveto != "" {
		err := ioutil.WriteFile(saveto, []byte(fmt.Sprintf("total queries: %d\nlog failed: %d\n", s.n, s.f)), 644)
		if err != nil {
			log.Printf("dump statistics to %s err: %s", saveto, err)
		}
	}
        s.l = s.n
}

func (s *Server) log2b(name string, addr net.Addr, backendIndex int) error {
	// trimsuffix and lowercase
	// name = strings.ToLower(strings.TrimSuffix(name, "."))
	clientip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return err
	}
	s.sysLog.Debug(fmt.Sprintf("query %s from %s", name, clientip))
	err = s.pools[backendIndex].Cmd("SETEX", name, 120, clientip).Err
	return err
}

// Log2b log quureies to backend by different channel/backend
func (s *Server) Log2b(backendIndex int, chanIndex int) {
	log.Printf("listening to backend %d channel %d" , backendIndex, chanIndex)
	for {
		query := <- s.logChan[backendIndex][chanIndex]
		err := s.log2b(query.name, query.addr, backendIndex)
		if err != nil {
			log.Printf("backend %d channel %d log2b %s %s raise err: %s", backendIndex, chanIndex, query.name, query.addr, err)
		}
	}
}
