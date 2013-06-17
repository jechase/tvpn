package main

import (
	"tvpn/ovpn"
	"flag"
)

func main() {
	port := flag.String("port","1234","It's a port!")
	flag.Parse()

	println(&ovpn.OVPN{RemoteIP: "1.2.3.4"})
	println(*port)
}