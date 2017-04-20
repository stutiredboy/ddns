package main

import (
	"os"
	"os/signal"
	"log"
	"time"
	"encoding/json"
	"syscall"
	"io/ioutil"

	"github.com/codegangsta/cli"
	"github.com/stutiredboy/ddns"
)

func main() {
	app := cli.NewApp()
	app.Name = "ddns"
	app.Usage = "DNS proxy for [D]etect Local [DNS] Server"
	app.Version = "0.0.3"
	app.Author, app.Email = "stutiredboy", "stutiredboy at gmail dot com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Usage:  "Load configuration from `FILE`",
		},
	}
	app.Action = func(c *cli.Context) {
		if c.String("config") == "" {
			log.Fatalf("ddns: config option is needed, read help first.")
		}
		config, err := ioutil.ReadFile(c.String("config"))
		if err != nil {
			log.Fatalf("File error: %v\n", err)
		}
		var conf ddns.Configurations
		err = json.Unmarshal(config, &conf)
		if err != nil {
			log.Fatalf("Unmarshal error: %v\n", err)
		}
		s, err := ddns.NewServer(conf)
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
		runPeriodically(s.Dump, conf.StatsPeriod, conf.StatsFile)
		log2b(s.Log2b, len(conf.Backends), conf.ChanNum)

		defer s.Shutdown() // in case of normal exit

		pid := os.Getpid();
		log.Printf("ddns: listening on %s with pid %d, proxying to %s", conf.Listen, pid, conf.NameServers)
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

// log query to backend
func log2b(handler func(int, int), backendnum int, channum int) {
	for i :=0 ; i < backendnum ; i++ {
		for j := 0 ; j < channum ; j++ {
			go func(i int, j int) {
				handler(i, j)
			}(i, j)
		}
	}
}

// do something periodically
func runPeriodically(handler func(int, string), period int, saveto string) {
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
