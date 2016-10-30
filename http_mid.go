package main

import (
	//	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	//	"regexp"
	"time"
	//  "reflect"
	//	"bufio"
)

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

func handleConnection(w http.ResponseWriter, r *http.Request) {
	client, _, err := w.(http.Hijacker).Hijack()
	log.Printf("read from client :%v\n", r.URL.String())

	if nil != err {
		log.Printf("client cant hijack\n")
		return
	}

	colonIndex := strings.Index(r.URL.Host, ":")
	var server net.Conn
	var errSer error
	if -1 == colonIndex {
		server, errSer = net.Dial("tcp", r.URL.Host+":80")
	} else {
		server, errSer = net.Dial("tcp", r.URL.Host)
	}
	log.Printf("connect the host %s\n", r.URL.Host)
	if errSer != nil {
		log.Printf("cant connect host : %s, and the err is :%v\n", r.URL.Host, errSer)
		return
	}
	defer server.Close()
	server.SetDeadline(time.Now().Add(time.Millisecond * 7000))

	if "CONNECT" == r.Method {
		_, e := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		if e != nil {
			fmt.Printf("Error to send message because of %s\n", e.Error())
			return
		}
	} else {
		r.Header.Del("Proxy-Connection")
		r.Header.Del("Connection")
		log.Printf("write to server  :%s", r.URL.String())
		if err = r.Write(server); nil != err {
			log.Printf("Error to send message because of %v\n", err)
			return
		}
	}
	err = pipe(client, server)
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
