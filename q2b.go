package ddns

import (
	"log"
	"net"
	"strings"
	"github.com/mediocregopher/radix.v2/pool"
)

func (s *Server) logq2b(name string, addr net.Addr, pool *pool.Pool) error {
	conn, err := pool.Get()
	if err != nil {
		log.Printf("q2b: %s", err)
		return err
	}
	defer pool.Put(conn)

	name = strings.TrimSuffix(name, ".")
	clientip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return err
	}
	err = conn.Cmd("SETEX", name, 120, clientip).Err
	if err != nil {
		log.Printf("q2b: %s", err)
	}
	return err
}
