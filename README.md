# `dnsp`: A DNS Proxy

Forked from: https://github.com/gophergala/dnsp

## INSTALL

    go get github.com/stutiredboy/ddns
    cd ${GOPATH}/stutiredboy/ddns/cmd/ddns
    go build

## Configurations

```
{
	"ConnectTimeout": 500,
	"Debug": true,
	"Backends": {
		"0": "127.0.0.1:6379",
		"1": "127.0.0.1:6379"
	},
	"StatsPeriod": 60,
	"StatsFile": "/home/tiredboy/ddns.stats",
	"NameServers": ["8.8.8.8", "8.8.4.4"],
	"ReadTimeout": 500,
	"ChanNum": 4,
	"PoolNum": 5,
	"Listen": "127.0.0.1:5353"
}
```

Configuration | Description
------ | ------
NameServers | name servers the DNS query forward to, format: address:port, default port is 53
Listen | UDP Listen address:port
StatsFile | stats file, absolute path
StatsPeriod | ddns dump stats periodically(seconds)
Backends | redis server
ConnectTimeout | timeout for connecting to redis server, Millisecond
ReadTimeout | timeout for read/write to redis server, Millisecond
ChanNum | concurrency numbers for writing to redis
PoolNum | redis connection pool, must little greater than *ChanNum*
Debug | verbose output
