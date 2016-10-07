package main

import (
	//	"bytes"
	"flag"
	"log"
	"net"
	"time"
)

type Sock5AuthType int

const (
	Sock5AuthTypeAnonymous Sock5AuthType = iota
	Sock5AuthTypeUsernamePwd
	Sock5AuthTypeAnonymousOrUsernamePwd
	Sock5AuthTypeUnkonwn
)

func getSock5AuthType(buf []byte) Sock5AuthType {
	if 3 == len(buf) {
		if 5 == buf[0] && 1 == buf[1] && 0 == buf[2] {
			return Sock5AuthTypeAnonymous
		}
		if 5 == buf[0] && 1 == buf[1] && 2 == buf[2] {
			return Sock5AuthTypeUsernamePwd
		}
		return Sock5AuthTypeUnkonwn
	}
	if 4 == len(buf) {
		if 5 == buf[0] && 2 == buf[1] && 0 == buf[2] && 2 == buf[3] {
			return Sock5AuthTypeAnonymousOrUsernamePwd
		}
	}
	return Sock5AuthTypeUnkonwn
}

type Sock5ConnType int

const (
	Sock5ConnTypeTcpDomain Sock5ConnType = iota
	Sock5ConnTypeTcpIp
	Sock5ConnTypeUdp
	Sock5ConnTypeUnkonwn
)

func getSock5ConnType(buf []byte) Sock5ConnType {
	if 4 <= len(buf) {
		if 5 == buf[0] && 1 == buf[1] && 0 == buf[2] && 3 == buf[3] {
			return Sock5ConnTypeTcpDomain
		}
		if 5 == buf[0] && 1 == buf[1] && 0 == buf[2] && 1 == buf[3] {
			return Sock5ConnTypeTcpIp
		}
		if 5 == buf[0] && 3 == buf[3] && 0 == buf[2] && 1 == buf[3] {
			return Sock5ConnTypeUdp
		}
	}
	return Sock5ConnTypeUnkonwn
}

func readFromConn(conn net.Conn) []byte {
	const HTTP_LEN = 20480
	const BUF_LEN = 1024

	buf := make([]byte, HTTP_LEN)
	buflen := 0
	tmp := make([]byte, BUF_LEN)

	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err.Error() != "EOF" {
				log.Print("io.error  ", err)
				return nil
			}
			break
		}

		if buflen+n > HTTP_LEN {
			log.Print("HTTP is too langer")
			return nil
		}
		if 0 != n {
			copy(buf[buflen:], tmp[:n])
			buflen += n
		}

		if n < BUF_LEN {
			break
		}
	}
	return buf[:buflen]
}

func getHost(curConnType Sock5ConnType, bufbody []byte) string {
	if Sock5ConnTypeTcpIp == curConnType {
	}
	if Sock5ConnTypeTcpDomain == curConnType {
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Millisecond * 5000))

	//auth
	buf := readFromConn(conn)
	if 0 == len(buf) {
		return
	}
	curAuthType := getSock5AuthType(buf)
	log.Printf("1.read from conn package %v, auth type is type %v\n", buf, curAuthType)
	if Sock5AuthTypeAnonymous != curAuthType {
		log.Printf("we cann't support the cur sock5authtype\n")
		return
	}
	w := []byte{0x05, 0x00}
	log.Printf("the test is %v", w)
	conn.Write([]byte{0x05, 0x00})

	//package
	bufbody := readFromConn(conn)
	if 0 == len(bufbody) {
		return
	}
	curConnType := getSock5ConnType(bufbody)
	t := (bufbody[0]).(type)
	log.Printf("isss t %v", t)
	log.Printf("2.from conn 4 bytes of package are %v, conn type if type %v \n", bufbody[:4], curConnType)
	if Sock5ConnTypeTcpDomain != curConnType || Sock5ConnTypeTcpIp != curConnType {
		log.Printf("cann't support the cur Sock5connyype\n")
		return
	}

	//create server conn
	host := getHost(curConnType, bufbody)
}

func main() {
	var host, port string
	flag.StringVar(&host, "h", "127.0.0.1", "which host do you want to listen")
	flag.StringVar(&port, "p", "6010", "which port do you want to listen")
	flag.Parse()

	ln, err := net.Listen("tcp", host+":"+port)
	log.Print("listen on ", host+":"+port)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print("get client connection error:", err)
			return
		}
		go handleConnection(conn)
	}
}
