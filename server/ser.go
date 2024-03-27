package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type Message struct {
	from    string
	payload []byte
}

type Server struct {
	listenAddr  string
	ln          net.Listener
	quitch      chan struct{}
	msgch       chan Message
	connections sync.Map
}

func NewServer(listenAddr string) *Server {
	return &Server{
		listenAddr: listenAddr,
		quitch:     make(chan struct{}),
		msgch:      make(chan Message, 10),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}

	s.ln = ln

	fmt.Printf("Сервер запущен!\nАдрес: %s\n", ln.Addr())

	go s.acceptLoop()

	<-s.quitch
	close(s.msgch)

	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("Ошибка подключения ", err)
			continue
		}

		fmt.Println("Новое подключение:", conn.RemoteAddr())

		s.connections.Store(conn, struct{}{})

		go s.readLoop(conn)
	}
}

func (s *Server) readLoop(conn net.Conn) {
	defer func() {
		conn.Close()
		s.connections.Delete(conn)
	}()

	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Ошибка чтения", err)
			return
		}

		s.msgch <- Message{
			from:    conn.RemoteAddr().String(),
			payload: buf[:n],
		}
	}
}

func (s *Server) Broadcast(msg []byte) {
	s.connections.Range(func(key, value interface{}) bool {
		conn := key.(net.Conn)
		_, err := conn.Write(msg)
		if err != nil {
			log.Println("Ошибка отправки сообщения:", err)
			s.connections.Delete(conn)
		}
		return true
	})
}

func main() {
	server := NewServer("127.0.0.1:8080")

	go func() {
		for msg := range server.msgch {
			fmt.Printf("(%s): %s\n", msg.from, string(msg.payload))
			server.Broadcast(msg.payload)
		}
	}()

	log.Fatal(server.Start())
}
