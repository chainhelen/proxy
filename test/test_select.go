package main

import (
	"log"
	"time"
)

func sum(n int, ch chan int) {
	t := time.NewTimer(time.Second * 5)
	select {
	case <-t.C:
		ch <- n * 100
	}
}

func main() {
	c1 := make(chan int, 2)
	c2 := make(chan int, 2)

	go sum(1, c1)
	go sum(2, c2)

	go func() {
		select {
		case a := <-c1:
			log.Printf("a %v", a)
		case b := <-c2:
			log.Printf("b %v", b)
		}
	}()
	log.Printf("ww\n")
}
