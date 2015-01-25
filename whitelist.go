package dnsp

import (
	"log"

	"github.com/miekg/dns"
)

const (
	Unknown host = iota
	White        // whitelisted
	Black        // blacklisted
)

type host uint8

type hosts map[string]host

// Whitelist whitelists a hosts.
func (s *Server) Whitelist(host string) {
	setHost(s.hosts, host, White)
}

// Blacklist blacklists a host.
func (s *Server) Blacklist(host string) {
	setHost(s.hosts, host, Black)
}

func setHost(hosts map[string]host, host string, b host) {
	if host == "" {
		return
	}
	if host[len(host)-1] != '.' {
		host += "."
	}
	hosts[host] = b
}

// IsAllowed returns whether we are allowed to resolve this host.
//
// If the server is whitelisting, the rusilt will be true if the host is on the whitelist.
// If the server is blacklisting, the result will be true if the host is NOT on the blacklist.
//
// NOTE: "host" must end with a dot.
func (s *Server) IsAllowed(host string) bool {
	log.Printf("%s %#v", host, s.hosts)
	b := s.hosts[host]
	if s.white {
		return b == White
	}
	return b != Black
}

func (s *Server) filter(qs []dns.Question) []dns.Question {
	result := []dns.Question{}
	for _, q := range qs {
		if s.IsAllowed(q.Name) {
			result = append(result, q)
		}
	}
	return result
}

func loadWhitelist(h hosts, path string) error {
	return nil
}

func loadBlacklist(h hosts, path string) error {
	return nil
}
