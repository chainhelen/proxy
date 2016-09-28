package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
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

func handleConnection(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v", r.Method)
	log.Printf("%v", r.URL.Host)

	log.Printf("connect the host %s\n", r.URL.Host)
	server, err := net.Dial("tcp", r.URL.Host)
	if err != nil {
		log.Printf("cant connect host : %s, and the err is :%v", r.URL.Host, err)
		return
	}
	defer server.Close()
	server.SetDeadline(time.Now().Add(time.Millisecond * 7000))

	if "CONNECT" == r.Method {
		_, e := w.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		if e != nil {
			fmt.Printf("Error to send message because of %s\n", e.Error())
			return
		} else {
			log.Printf("---- %v", r.Method)
			//			r.Header.Del("Proxy-Connection")
			//			r.Header.Del("Connection")
		}
	} else {
	}
}

func main() {
	var host, port string
	flag.StringVar(&host, "h", "127.0.0.1", "which host do you want to listen")
	flag.StringVar(&port, "p", "6010", "which port do you want to listen")
	flag.Parse()

	server := &http.Server{
		Addr:         host + ":" + port,
		Handler:      http.HandlerFunc(handleConnection),
		ReadTimeout:  1 * time.Hour,
		WriteTimeout: 1 * time.Hour,
	}

	log.Print("listen on ", host+":"+port)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
