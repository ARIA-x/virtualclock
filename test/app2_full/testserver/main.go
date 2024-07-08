package main

import (
	"fmt"
	"io/ioutil"
	"net"
)

func main() {
	tempDir, err := ioutil.TempDir("", "golang-sample-echo-server.")
	socket := tempDir + "/testserver"
	fmt.Printf("%s\n", socket)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		fmt.Printf(err.Error())
	}
	defer listener.Close()

	fmt.Println("server launched...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Print(err.Error())
		}

		fmt.Println(">>> accepted")
		go server(conn)
	}
}

func server(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("Connected: %s\n", conn.RemoteAddr().Network())

	buf := make([]byte, 1024)
	for {
		nr, err := conn.Read(buf)
		if err != nil {
			fmt.Print(err.Error())
			return
		}
		data := buf[0:nr]
		fmt.Printf("Received : %v", string(data))
	}
	/*
			_, err = conn.Write(data)
			if err != nil {
				fmt.Print(err.Error())
			}
		conn.Close()
	*/

}
