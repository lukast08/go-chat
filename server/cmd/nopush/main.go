package main

import (
	"fmt"
	"net"
)

func main() {
	udpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("127.0.0.1", "62457"))
	if err != nil {
		panic(err)
	}

	conn, err := net.DialTCP("tcp", nil, udpAddr)
	if err != nil {
		panic(err)
	}

	l, err := conn.Write([]byte("Hiii!!!"))
	if err != nil {
		panic(err)
	}

	fmt.Println("written:", l)
}
