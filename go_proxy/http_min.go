package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"regexp"
	"time"
	//  "reflect"
	//	"bufio"
)

func getRemoteInfoFromHeader(buf []byte) ([]byte, []byte) {
	//find the lineBuf start
	hostStart := bytes.Index(buf, []byte("Host:"))
	if -1 == hostStart {
		return []byte(""), []byte("")
	}
	hostStart += 6

	//find the lineBuf end
	lineBuf := buf[hostStart:]
	hostEnd := bytes.Index(lineBuf, []byte("\r\n"))
	if -1 == hostEnd {
		return []byte(""), []byte("")
	}

	//get the lineBuf
	lineBuf = lineBuf[:hostEnd]

	//get the address port
	portIndex := bytes.Index(lineBuf, []byte(":"))
	if -1 != portIndex {
		address := make([]byte, len(lineBuf[:portIndex]))
		copy(address, lineBuf[:portIndex])

		port := make([]byte, len(lineBuf[(portIndex+1):]))
		copy(port, lineBuf[(portIndex+1):])

		return address, port
	} else {
		address := make([]byte, len(lineBuf))
		copy(address, lineBuf)

		port := make([]byte, len([]byte("80")))
		copy(port, []byte("80"))

		return address, port
	}
	return []byte(""), []byte("")
}

func fwData(sock1, sock2 net.Conn, p, connOpenFlag []bool) {

	defer func() {
		if false == p[0] && false == p[1] {
			connOpenFlag[0] = false
			connOpenFlag[1] = false
		}
	}()

	const BUF_LEN = 2048
	tmp := make([]byte, BUF_LEN)
	pipeflag := true

	for {
		if !pipeflag {
			return
		}
		n, rerr := sock1.Read(tmp)
		if rerr != nil {
			if rerr.Error() != "EOF" {
				log.Print("io.read err ", rerr)
				return
			}
			pipeflag = false
		} else {
			if 0 != n {
				_, werr := sock2.Write(tmp[:n])
				if werr != nil {
					log.Print("io.write err %v", werr)
					return
				}
			} else {
				pipeflag = false
			}
		}
	}
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

func changeUrlToPathInHeader(buf []byte) []byte {
	reg, err := regexp.Compile("(http://[^/]+|Keep-Alive:.+\r\n|Proxy-Connection:.+\r\n)|Connection:.+\r\n")
	if nil != err {
		log.Print("reg is not right", err)
		return []byte("")
	}
	rep := []byte("")
	tmp := reg.ReplaceAll(buf, rep)

	reg2, err2 := regexp.Compile("\r\n\r\n")
	if nil != err {
		log.Print("reg2, is not right", err2)
	}
	rep2 := []byte("\r\nConnection:close\r\n\r\n")

	return reg2.ReplaceAll(tmp, rep2)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Millisecond * 5000))

	connOpenFlag := []bool{true, true}

	buf := readFromConn(conn)
	if 0 == len(buf) {
		return
	}
	log.Printf("read from client :package len = %d \t%s\n", len(buf), buf[:bytes.Index(buf, []byte("\r\n"))])

	//find the header
	bodyPos := bytes.Index(buf, []byte("\r\n\r\n"))
	if -1 == bodyPos {
		log.Printf("bodyPos  failed")
		return
	}
	header := make([]byte, len(buf[:bodyPos]))
	copy(header, buf[:bodyPos])

	//find the remote address port
	address, port := getRemoteInfoFromHeader(buf)
	if 0 == len(address) || 0 == len(port) {
		log.Printf("cant find the address or port")
		return
	}

	//connect to the server
	host := append(append(address, ':'), port...)
	log.Printf("connect the host %s\n", string(host))
	server, err := net.Dial("tcp", string(host))
	if err != nil {
		log.Printf("cant connect host : %s, and the err is :%v", string(host), err)
		return
	}

	defer server.Close()
	server.SetDeadline(time.Now().Add(time.Millisecond * 7000))

	go fwData(server, conn, []bool{false, false}, connOpenFlag)

	// if the method named "connect"
	CONNECT := []byte("CONNECT")
	HEADER_F7 := buf[:7]
	if bytes.Equal(CONNECT, HEADER_F7) {
		_, e := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		if e != nil {
			fmt.Printf("Error to send message because of %s\n", e.Error())
			return
		}
		go fwData(conn, server, []bool{true, true}, connOpenFlag)
	} else {
		chBuf := changeUrlToPathInHeader(buf)
		log.Printf("write to server  :package len = %d \t%s\n", len(chBuf), chBuf[:bytes.Index(chBuf, []byte("\r\n"))])
		go server.Write(chBuf)
	}

	for {
		time.Sleep(500 * time.Millisecond)
		if false == connOpenFlag[0] && false == connOpenFlag[1] {
			return
		}
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
