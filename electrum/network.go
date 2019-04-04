package electrum

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// ClientVersion ...
	ClientVersion = "go-electrum1.0"

	// ProtocolVersion ...
	ProtocolVersion = "1.4"

	connTimeout = 30 * time.Second
	nl          = byte('\n')
)

var (
	// DebugMode ...
	DebugMode bool

	// ErrServerConnected ...
	ErrServerConnected = errors.New("server is already connected")

	// ErrServerShutdown ...
	ErrServerShutdown = errors.New("server has shutdown")

	// ErrTimeout ...
	ErrTimeout = errors.New("request timeout")

	// ErrNotImplemented ...
	ErrNotImplemented = errors.New("API call is not implemented")
)

// Transport ...
type Transport interface {
	SendMessage([]byte) error
	Responses() <-chan []byte
	Errors() <-chan error
}

// TCPTransport ...
type TCPTransport struct {
	conn      net.Conn
	responses chan []byte
	errors    chan error
}

// NewTCPTransport ...
func NewTCPTransport(addr string) (*TCPTransport, error) {
	conn, err := net.DialTimeout("tcp", addr, connTimeout)
	if err != nil {
		return nil, err
	}

	tcp := &TCPTransport{
		conn:      conn,
		responses: make(chan []byte),
		errors:    make(chan error),
	}

	go tcp.listen()

	return tcp, nil
}

// NewSSLTransport ...
func NewSSLTransport(addr string, config *tls.Config) (*TCPTransport, error) {
	dialer := net.Dialer{
		Timeout: connTimeout,
	}
	conn, err := tls.DialWithDialer(&dialer, "tcp", addr, config)
	if err != nil {
		return nil, err
	}

	tcp := &TCPTransport{
		conn:      conn,
		responses: make(chan []byte),
		errors:    make(chan error),
	}

	go tcp.listen()

	return tcp, nil
}

func (t *TCPTransport) listen() {
	defer t.conn.Close()
	reader := bufio.NewReader(t.conn)

	for {
		line, err := reader.ReadBytes(nl)
		if err != nil {
			t.errors <- err
			break
		}
		if DebugMode {
			log.Printf("%s [debug] %s -> %s", time.Now().Format("2006-01-02 15:04:05"), t.conn.RemoteAddr(), line)
		}

		t.responses <- line
	}
}

// SendMessage ...
func (t *TCPTransport) SendMessage(body []byte) error {
	if DebugMode {
		log.Printf("%s [debug] %s <- %s", time.Now().Format("2006-01-02 15:04:05"), t.conn.RemoteAddr(), body)
	}

	_, err := t.conn.Write(body)
	return err
}

// Responses ...
func (t *TCPTransport) Responses() <-chan []byte {
	return t.responses
}

// Errors ...
func (t *TCPTransport) Errors() <-chan error {
	return t.errors
}

type container struct {
	content []byte
	err     error
}

// Server ...
type Server struct {
	transport Transport

	handlers     map[uint64]chan *container
	handlersLock sync.RWMutex

	pushHandlers     map[string][]chan *container
	pushHandlersLock sync.RWMutex

	Error chan error
	quit  chan struct{}

	nextID uint64
}

// NewServer ...
func NewServer() *Server {
	s := &Server{
		handlers:     make(map[uint64]chan *container),
		pushHandlers: make(map[string][]chan *container),

		Error: make(chan error),
		quit:  make(chan struct{}),
	}

	return s
}

// ConnectTCP ...
func (s *Server) ConnectTCP(addr string) error {
	if s.transport != nil {
		return ErrServerConnected
	}

	transport, err := NewTCPTransport(addr)
	if err != nil {
		return err
	}

	s.transport = transport
	go s.listen()

	return nil
}

// ConnectSSL ...
func (s *Server) ConnectSSL(addr string, config *tls.Config) error {
	if s.transport != nil {
		return ErrServerConnected
	}

	transport, err := NewSSLTransport(addr, config)
	if err != nil {
		return err
	}

	s.transport = transport
	go s.listen()

	return nil
}

type apiErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *apiErr) Error() string {
	return fmt.Sprintf("errNo: %d, errMsg: %s", e.Code, e.Message)
}

type response struct {
	ID     uint64  `json:"id"`
	Method string  `json:"method"`
	Error  *apiErr `json:"error"`
}

func (s *Server) listen() {
	for {
		select {
		case err := <-s.transport.Errors():
			s.Error <- err
			s.shutdown()
		case bytes := <-s.transport.Responses():
			result := &container{
				content: bytes,
			}

			msg := &response{}
			err := json.Unmarshal(bytes, msg)
			if err != nil {
				if DebugMode {
					log.Printf("Unmarshal received message failed: %v", err)
				}
				result.err = fmt.Errorf("Unmarshal received message failed: %v", err)
			} else if msg.Error != nil {
				result.err = msg.Error
			}

			if len(msg.Method) > 0 {
				s.pushHandlersLock.RLock()
				handlers := s.pushHandlers[msg.Method]
				s.pushHandlersLock.RUnlock()

				for _, handler := range handlers {
					select {
					case handler <- result:
					default:
					}
				}
			}

			s.handlersLock.RLock()
			c, ok := s.handlers[msg.ID]
			s.handlersLock.RUnlock()

			if ok {
				c <- result
			}
		}
	}
}

func (s *Server) listenPush(method string) <-chan *container {
	c := make(chan *container, 1)
	s.pushHandlersLock.Lock()
	s.pushHandlers[method] = append(s.pushHandlers[method], c)
	s.pushHandlersLock.Unlock()

	return c
}

type request struct {
	ID     uint64        `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

func (s *Server) request(method string, params []interface{}, v interface{}) error {
	select {
	case <-s.quit:
		return ErrServerShutdown
	default:
	}

	msg := request{
		ID:     atomic.AddUint64(&s.nextID, 1),
		Method: method,
		Params: params,
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	bytes = append(bytes, nl)

	err = s.transport.SendMessage(bytes)
	if err != nil {
		return err
	}

	c := make(chan *container, 1)

	s.handlersLock.Lock()
	s.handlers[msg.ID] = c
	s.handlersLock.Unlock()

	var resp *container
	select {
	case resp = <-c:
	case <-time.After(connTimeout):
		return ErrTimeout
	}

	if resp.err != nil {
		return resp.err
	}

	s.handlersLock.Lock()
	delete(s.handlers, msg.ID)
	s.handlersLock.Unlock()

	if v != nil {
		err = json.Unmarshal(resp.content, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) shutdown() {
	close(s.quit)

	s.transport = nil
	s.handlers = nil
	s.pushHandlers = nil
}
