package main

import (
	//	"bytes"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"strconv"
	//	"time"
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

func getHostPortAndCutBufbodyDown(curConnType Sock5ConnType, bufbody []byte) string {
	host := ""
	if Sock5ConnTypeTcpIp == curConnType {
		host = string(net.IPv4(bufbody[0], bufbody[1], bufbody[2], bufbody[3]))
		log.Printf("getHost the if is %v\n", host)
	}
	if Sock5ConnTypeTcpDomain == curConnType {
		len := bufbody[0]
		host = string(bufbody[1 : len+1])
		host += ":"
		port := binary.BigEndian.Uint32(append([]byte{0, 0}, bufbody[len+1:len+3]...))
		host += strconv.FormatUint(uint64(port), 10)

		//cutdown
		bufbody = bufbody[len+3:]
	}
	return host
}

func writeTo(from, to net.Conn, errch chan error) error {
	_, err := io.Copy(from, to)
	return err
}

func pipe(a, b net.Conn) error {
	errch := make(chan error, 2)

	go writeTo(a, b, errch)
	go writeTo(b, a, errch)

	err1, err2 := <-errch, <-errch

	if nil != err1 {
		log.Printf("pipe err type 1 : %v\n", err1)
	}

	if nil != err2 {
		log.Printf("pipe err type 2 : %v\n", err1)
	}

	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	//	conn.SetDeadline(time.Now().Add(time.Millisecond * 1000))

	//auth
	buf := readFromConn(conn)
	if 0 == len(buf) {
		return
	}
	curAuthType := getSock5AuthType(buf)
	log.Printf("1.\tread from conn package %v, auth type is type %v\n", buf, curAuthType)
	if Sock5AuthTypeAnonymous != curAuthType {
		log.Printf("\t\twe cann't support the cur sock5authtype\n")
		return
	}
	conn.Write([]byte{0x05, 0x00})

	//package
	bufbody := readFromConn(conn)
	if 0 == len(bufbody) {
		return
	}
	curConnType := getSock5ConnType(bufbody)

	//t := (bufbody[0]).(type)
	//log.Printf("isss t %v", t)
	log.Printf("2.\tfrom conn 4 bytes of package are %v, conn type if type %v \n", bufbody[:4], curConnType)

	if Sock5ConnTypeTcpDomain != curConnType && Sock5ConnTypeTcpIp != curConnType {
		log.Printf("cann't support the cur Sock5connyype\n")
		return
	}

	//create server conn
	bufbody = bufbody[4:]
	log.Printf("\t\tget the body %v\n", bufbody)
	hostAndPort := getHostPortAndCutBufbodyDown(curConnType, bufbody)
	log.Printf("\t\tget the hostAndPort %s\n", hostAndPort)

	server, errSer := net.Dial("tcp", hostAndPort)
	if errSer != nil {
		log.Printf("\t\tcant connect host : %s, and the err is :%v\n", hostAndPort, errSer)
		return
	}
	log.Printf("\t\tconnect server host : %s ok", hostAndPort)
	defer server.Close()
	//	server.SetDeadline(time.Now().Add(time.Millisecond * 1000))
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x7f, 0x00, 0x00, 0x01, 0x17, 0x7a})

	err := pipe(conn, server)
	if err != nil {
		log.Printf("Error pipe because of %s\n", err.Error())
		return
	}

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
