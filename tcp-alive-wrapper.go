package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
)

var now int32 = 0
var counter int32 = 0

func handleClient(conn net.Conn, done chan interface{}, m *sync.Mutex) {
	m.Lock()
	id := counter
	counter++
	now++
	log.Printf("ID: %d open, Exists: %d", id, now)
	conn.Write([]byte(fmt.Sprintf("%d\r\n", now)))
	m.Unlock()

	defer func() {
		m.Lock()
		now--
		log.Printf("ID: %d close, Exists: %d", id, now)
		m.Unlock()
	}()

	notify := make(chan error, 1)
	go func() {
		r := bufio.NewReader(conn)
		for {
			_, _, err := r.ReadLine()
			if err != nil {
				log.Printf("catch error: %v", err)
				notify <- err
				return
			}
		}
	}()

	for {
		select {
		case err := <-notify:
			if io.EOF == err {
				log.Printf("error client")
				return
			}
		case <-done:
			log.Printf("end client")
			return
		}
	}
}

func Start(done chan interface{}, wg *sync.WaitGroup) {
	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}
	defer listener.Close()

	for {
		m := new(sync.Mutex)
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Fatal: %v", err)
		}

		go func() {
			wg.Add(1)
			handleClient(conn, done, m)
			conn.Close()
			wg.Done()
		}()
	}
}

func main() {
	wg := &sync.WaitGroup{}
	done := make(chan interface{})
	go Start(done, wg)

	flag.Usage = func() {}
	flag.Parse()

	cmd := flag.Args()
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout

	err := c.Start()
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}
	err = c.Wait()
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}

	close(done)
	wg.Wait()
}
