package main

import (
	"flag"
	"fmt"
	"github.com/AdRoll/hologram/protocol"
	"net"
)

var host = flag.String("host", "localhost", "IP or hostname to ping")
var port = flag.Int("port", 3100, "Port to connect to for ping")

func main() {
	flag.Parse()
	connString := fmt.Sprintf("%s:%d", *host, *port)

	conn, err := net.Dial("tcp", connString)
	if err != nil {
		panic(err)
	}

	fmt.Printf("sending ping to %s...\n", connString)
	err = protocol.Write(conn, &protocol.Message{Ping: &protocol.Ping{}})
	response, err := protocol.Read(conn)

	if err != nil {
		panic(err)
	}

	if response.GetPing() != nil {
		fmt.Println("Got pong!")
	}
}
