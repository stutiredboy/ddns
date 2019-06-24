package ddns

import (
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"net"
	"strings"
	"time"

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
	c     *dns.Client
	s     *dns.Server
	pools map[int]*pool.Pool
	/* current queries counter */
	currQueries int64
	/* last queries counter for qps */
	lastQueries int64
	/* current failed counter */
	currFailed int64
	/* last failed counter */
	lastFailed int64
	/* failedRate */
	failedRate  float64
	sysLog      *syslog.Writer
	logChan     map[int]map[int]chan qinfo
	lenBackends int
	ExpiresIn   int
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
		for i := 0; i < c.ChanNum; i++ {
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
		pools:       pools,
		currQueries: 0,
		lastQueries: 0,
		currFailed:  0,
		lastFailed:  0,
		failedRate:  0.0,
		sysLog:      sysLog,
		logChan:     logChan,
		lenBackends: len(c.Backends),
		ExpiresIn:   c.ExpiresIn,
	}

	s.s.Handler = dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		// If no upstream proxy is present, drop the query:
		if len(c.NameServers) == 0 {
			log.Printf("no nameservers, drop query")
			dns.HandleFailed(w, r)
			return
		}
		isEdns0 := true
		ecs := GetEdns0Subnet(r)
		if ecs == nil {
			isEdns0 = false
			ecs = SetEdns0Subnet(r, w.RemoteAddr())
		}
		if c.Debug {
			log.Printf("query %+v from %s msg %+v with ecs %s", r.Question, w.RemoteAddr(), r.MsgHdr, ecs.String())
		}
		/* r == nil:
		panic: runtime error: invalid memory address or nil pointer dereference
		*/
		if r == nil {
			log.Printf("dns Msg is nil, ignore it.")
			return
		}
		if r.Question == nil || len(r.Question) == 0 {
			log.Printf("no query Question, drop query")
			dns.HandleFailed(w, r)
			return
		}
		// send query info to channel
		name := strings.ToLower(strings.TrimSuffix(r.Question[0].Name, "."))
		// Backend and Channel must use different hash method
		backendIndex := backendHash(name) % s.lenBackends
		chanIndex := channelHash(name) % c.ChanNum
		select {
		case s.logChan[backendIndex][chanIndex] <- qinfo{name, w.RemoteAddr()}:
		default:
			s.currFailed++
			log.Printf("receive query %s %s, but backend %d channel%d full", r.Question[0].Name, w.RemoteAddr(), backendIndex, chanIndex)
		}
		// increase queries counter
		s.currQueries++

		// Proxy Query:
		for _, addr := range c.NameServers {
			in, _, err := s.c.Exchange(r, addr)
			if err != nil {
				continue
			}
			if !isEdns0 {
				RemoveEdns0Subnet(in)
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
	qps := (s.currQueries - s.lastQueries) / int64(period)
	if qps > 0 {
		s.failedRate = float64(s.currFailed-s.lastFailed) / float64(s.currQueries-s.lastQueries)
	}
	log.Printf("total queries: %d, qps: %d, log failed: %d, failed rate: %f", s.currQueries, qps, s.currFailed, s.failedRate)
	if saveto != "" {
		err := ioutil.WriteFile(saveto, []byte(fmt.Sprintf("total queries: %d\nlog failed: %d\nfailed rate: %f", s.currQueries, s.currFailed, s.failedRate)), 644)
		if err != nil {
			log.Printf("dump statistics to %s err: %s", saveto, err)
		}
	}
	s.lastQueries = s.currQueries
	s.lastFailed = s.currFailed
}

func (s *Server) log2b(name string, addr net.Addr, backendIndex int) error {
	// trimsuffix and lowercase
	// name = strings.ToLower(strings.TrimSuffix(name, "."))
	clientip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return err
	}
	s.sysLog.Debug(fmt.Sprintf("query %s from %s", name, clientip))
	err = s.pools[backendIndex].Cmd("SETEX", name, s.ExpiresIn, clientip).Err
	return err
}

// Log2b log quureies to backend by different channel/backend
func (s *Server) Log2b(backendIndex int, chanIndex int) {
	log.Printf("listening to backend %d channel %d", backendIndex, chanIndex)
	for {
		query := <-s.logChan[backendIndex][chanIndex]
		err := s.log2b(query.name, query.addr, backendIndex)
		if err != nil {
			log.Printf("backend %d channel %d log2b %s %s raise err: %s", backendIndex, chanIndex, query.name, query.addr, err)
		}
	}
}

// GetEdns0Subnet get ecs from query msg
func GetEdns0Subnet(query *dns.Msg) net.IP {
	opt := query.IsEdns0()
	if opt == nil {
		return nil
	}
	for _, s := range opt.Option {
		switch e := s.(type) {
		case *dns.EDNS0_SUBNET:
			return e.Address
		}
	}
	return nil
}

// SetEdns0Subnet append client subnet to dns query
func SetEdns0Subnet(query *dns.Msg, addr net.Addr) net.IP {
	// append EDNS0_SUBNET to query
	var remoteAddr net.IP
	// 1 for IPv4, 2 for IPv6
	var addrFamily uint16
	// 32 for IPv4, 128 for IPv6
	var netMask uint8

	if addr.(*net.UDPAddr).IP.To16() != nil && addr.(*net.UDPAddr).IP.To4() == nil {
		addrFamily = 2
		netMask = 128
		remoteAddr = addr.(*net.UDPAddr).IP.To16()
	} else {
		addrFamily = 1
		netMask = 32
		remoteAddr = addr.(*net.UDPAddr).IP.To4()
	}
	o := new(dns.OPT)
	o.Hdr.Name = "."
	o.Hdr.Rrtype = dns.TypeOPT
	e := new(dns.EDNS0_SUBNET)
	e.Code = dns.EDNS0SUBNET
	// 1 for IPv4 source addr, 2 for IPv6 source addr
	e.Family = addrFamily
	// 32 for IPv4, 128 for IPv6
	e.SourceNetmask = netMask
	e.SourceScope = 0
	e.Address = remoteAddr
	// have EDNS0 Option already
	for i := len(query.Extra) - 1; i >= 0; i-- {
		if query.Extra[i].Header().Rrtype == dns.TypeOPT {
			opt := query.Extra[i].(*dns.OPT)
			opt.Option = append(opt.Option, e)
			query.Extra = append(query.Extra[:i], query.Extra[i+1:]...)
			query.Extra = append(query.Extra, opt)
			return e.Address
		}
	}
	// client query without EDNS0 Option
	o.Option = append(o.Option, e)
	query.Extra = append(query.Extra, o)
	return e.Address
}

// RemoveEdns0Subnet remove EDNS Subnet from answer section
func RemoveEdns0Subnet(answer *dns.Msg) error {
	// RFC 6891, Section 6.1.1 allows the OPT record to appear
	// anywhere in the additional record section, but it's usually at
	// the end so start there.
	for i := len(answer.Extra) - 1; i >= 0; i-- {
		if answer.Extra[i].Header().Rrtype == dns.TypeOPT {
			// opt := answer.Extra[i].(*dns.OPT)
			// answer.Extra = append(answer.Extra[:i], answer.Extra[i+1:]...)
			opt := answer.Extra[i].(*dns.OPT)
			for oi := len(opt.Option) - 1; oi >= 0; oi-- {
				switch opt.Option[oi].(type) {
				case *dns.EDNS0_SUBNET:
					opt.Option = append(opt.Option[:oi], opt.Option[oi+1:]...)
					answer.Extra = append(answer.Extra[:i], answer.Extra[i+1:]...)
					answer.Extra = append(answer.Extra, opt)
					return nil
				}
			}
		}
	}
	return nil
}
