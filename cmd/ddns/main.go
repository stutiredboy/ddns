package main

import (
	"os"
	"os/signal"
	"log"
	"time"
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
	app.Usage = "DNS proxy for [D]etect Local [DNS] Server"
	app.Version = "0.0.1"
	app.Author, app.Email = "stutiredboy", "stutiredboy at gmail dot com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "listen, l",
			Value:  "127.0.0.1:53",
			Usage:  "listen address (host:port, host or :port)",
		},
		cli.StringFlag{
			Name:   "resolve, r",
			Value:  DefaultResolve,
			Usage:  "comma-separated list of name servers (host:port or host)",
		},
		// period for dump useful information, such as total query and qps
		cli.IntFlag{
			Name: "period",
			Value: 60,
			Usage: "time to dump qps (int seconds)",
		},
		cli.StringFlag{
			Name:   "statsfile",
			Usage:  "periodically save mem to statsfile, need abs path",
		},
		cli.StringFlag{
			Name: "backend, b",
			Value: "127.0.0.1:6379",
			Usage: "redis backend address (host:port)",
		},
		cli.IntFlag{
			Name: "poolnum",
			Value: 10,
			Usage: "redis backend connection pool size (int)",
		},
		cli.IntFlag{
			Name: "connect-timeout",
			Value: 1000,
			Usage: "redis connection create timeout (int milliseconds)",
		},
		cli.IntFlag{
			Name: "read-timeout",
			Value: 100,
			Usage: "redis connection write/read timeout (int milliseconds)",
		},
		cli.BoolFlag{
			Name: "debug, d",
			Usage: "debug mode, logger verbosely",
		},
	}
	app.Action = func(c *cli.Context) {
		resolve := []string{}
		if res := c.String("resolve"); res != "false" && res != "" {
			resolve = strings.Split(res, ",")
		}
		poolnum := c.Int("poolnum")
		if poolnum == 0 {
			poolnum = 10
		}
		channum := 0
		if poolnum > 2 {
			channum = poolnum / 2
		} else {
			channum = poolnum
		}
		o := &ddns.Options{
			Bind:      c.String("listen"),
			Resolve:   resolve,
			Backend:   c.String("backend"),
			PoolNum:   poolnum,
			ChanNum:   channum,
			ConnectTimeout:   c.Int("connect-timeout"),
			ReadTimeout:   c.Int("read-timeout"),
			Debug:     c.Bool("debug"),
		}
		s, err := ddns.NewServer(*o)
		if err != nil {
			log.Fatalf("ddns: %s", err)
		}

		catch(func(sig os.Signal) int {
			os.Stderr.Write([]byte{'\r'})
			log.Printf("ddns: shutting down by signal <%s>", sig)
			s.Shutdown()
			return 0
		}, syscall.SIGINT, syscall.SIGTERM)

		// log query counter periodically
		run_periodically(s.Dump, c.Int("period"), c.String("statsfile"))
		log2b(s.Log2b, channum)

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

func log2b(handler func(int), channum int) {
	for i := 0 ; i < channum ; i++ {
		go func(i int) {
			handler(i)
		}(i)
	}
}

// do something periodically
func run_periodically(handler func(int, string), period int, saveto string) {
	ticker := time.NewTicker(time.Duration(period) * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
				case <- ticker.C:
					handler(period, saveto)
				case <- quit:
					ticker.Stop()
					return
			}
		}
	}()
}
