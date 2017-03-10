package ddns

import (
	"log"
	"net"
	"github.com/mediocregopher/radix.v2/pool"
)

func (s *Server) logq2b(name string, addr net.Addr, pool *pool.Pool) error {
	conn, err := pool.Get()
	if err != nil {
		log.Printf("q2b: %s", err)
		return err
	}
	defer pool.Put(conn)

	err = conn.Cmd("SETEX", name, 120, addr).Err
	if err != nil {
		log.Printf("q2b: %s", err)
	}
	return err
}
