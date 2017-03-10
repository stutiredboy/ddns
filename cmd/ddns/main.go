package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/stutiredboy/ddns"
)

// DefaultResolve is the default list of nameservers for the `--resolve` flag.
var DefaultResolve = "8.8.4.4,8.8.8.8"

func main() {
	app := cli.NewApp()
	app.Name = "ddns"
	app.Usage = "DNS proxy for DNS Detect"
	app.Version = "0.0.1"
	app.Author, app.Email = "", ""
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "listen, l",
			Value:  "127.0.0.1:53",
			Usage:  "listen address (host:port, host or :port)",
			EnvVar: "DDNS_BIND",
		},
		cli.StringFlag{
			Name:   "resolve, r",
			Value:  DefaultResolve,
			Usage:  "comma-separated list of name servers (host:port or host)",
			EnvVar: "DDNS_SERVER",
		},
		cli.StringFlag{
			Name: "backend, b",
			Value: "localhost:6379",
			Usage: "redis backend address (host:port)",
			EnvVar: "DDNS_BACKEND",
		},
		cli.IntFlag{
			Name: "poolnum",
			Value: 10,
			Usage: "redis backend connection pool size (int)",
			EnvVar: "DDNS_POOLNUM",
		},
	}
	app.Action = func(c *cli.Context) {
		resolve := []string{}
		if res := c.String("resolve"); res != "false" && res != "" {
			resolve = strings.Split(res, ",")
		}
		o := &ddns.Options{
			Bind:      c.String("listen"),
			Resolve:   resolve,
			Backend:   c.String("backend"),
			PoolNum:   c.Int("poolnum"),
		}
		s, err := ddns.NewServer(*o)
		if err != nil {
			log.Fatalf("ddns: %s", err)
		}

		catch(func(sig os.Signal) int {
			os.Stderr.Write([]byte{'\r'})
			log.Printf("ddns: shutting down")
			s.Shutdown()
			return 0
		}, syscall.SIGINT, syscall.SIGTERM)
		defer s.Shutdown() // in case of normal exit

		pid := os.Getpid();
		if len(o.Resolve) == 0 {
			log.Printf("ddns: listening on %s with pid %d", o.Bind, pid)
		} else {
			log.Printf("ddns: listening on %s with pid %d, proxying to %s", o.Bind, pid, o.Resolve)
		}
		if err := s.ListenAndServe(); err != nil {
			log.Fatalf("ddns: %s", err)
		}
	}
	app.Run(os.Args)
}

// catch handles system calls using the given handler function.
func catch(handler func(os.Signal) int, signals ...os.Signal) {
	c := make(chan os.Signal, 1)
	for _, s := range signals {
		signal.Notify(c, s)
	}
	go func() {
		os.Exit(handler(<-c))
	}()
}
