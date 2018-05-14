package web

import (
	"github.com/gorilla/websocket"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    4096,
	WriteBufferSize:   4096,
	EnableCompression: true,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}

	_, addr, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}

	s.socketLock.Lock()
	s.openSockets[string(addr)] = conn
	s.socketLock.Unlock()

	go waitForDisconnect(conn, string(addr), s.disconnectChan)
}

func (s *Server) ProcessSocketRequests() {
	for {
		select {
		case addr := <-s.disconnectChan:
			s.socketLock.RLock()
			_, ok := s.openSockets[addr]
			s.socketLock.RUnlock()
			if ok {
				s.socketLock.Lock()
				delete(s.openSockets, addr)
				s.socketLock.Unlock()
			}
		case addr := <-s.addrChan:
			s.socketLock.RLock()
			conn, ok := s.openSockets[addr]
			s.socketLock.RUnlock()
			if ok {
				err := conn.WriteMessage(1, []byte(`{"paymentReceived": true}`))
				if err != nil {
					log.Error(err)
				}
				s.socketLock.Lock()
				delete(s.openSockets, addr)
				s.socketLock.Unlock()
			}
		case <-s.ctx.Done():
			break
		}
	}
}

func waitForDisconnect(conn *websocket.Conn, addr string, disconnectChan chan string) {
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	disconnectChan <- addr
	conn.Close()
}
