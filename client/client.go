package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	connection, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		panic(err)
	}
	defer func() {
		err := connection.Close()
		if err != nil {
			log.Println("Failed to close connection", err)
		}
	}()

	go func() {
		for {
			buff := make([]byte, 1024)
			n, err := connection.Read(buff)
			if err != nil {
				log.Fatalln("Error reading:", err.Error())
				return
			}

			if n > 0 {
				fmt.Print(string(buff[:n]))
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		str, _ := reader.ReadString('\n')

		if _, err := connection.Write([]byte(str)); err != nil {
			return
		}
	}
}
