package smtpd

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hownowstephen/email"
)

type Server struct {
	Name string

	TLSConfig  *tls.Config
	ServerName string

	// MaxSize of incoming message objects, zero for no cap otherwise
	// larger messages are thrown away
	MaxSize int

	// MaxConn limits the number of concurrent connections being handled
	MaxConn int

	// MaxCommands is the maximum number of commands a server will accept
	// from a single client before terminating the session
	MaxCommands int

	// RateLimiter gets called before proceeding through to message handling
	RateLimiter func(*Conn) bool

	// Handler is the handoff function for messages
	Handler MessageHandler
}

// NewServer creates a server with the default settings
func NewServer(handler func(*email.Message) error) *Server {
	name, err := os.Hostname()
	if err != nil {
		name = "localhost"
	}
	return &Server{
		Name:        name,
		ServerName:  name,
		MaxSize:     131072,
		MaxCommands: 100,
		Handler:     handler,
	}
}

// UseTLS tries to enable TLS on the server (can also just explicitly set the TLSConfig)
func (s *Server) UseTLS(cert, key string) error {
	c, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("Could not load TLS keypair, %v", err)
	}
	s.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{c},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		Rand:         rand.Reader,
	}
	return nil
}

// ListenAndServe starts listening for SMTP commands at the supplied TCP address
func (s *Server) ListenAndServe(addr string) error {

	// Start listening for SMTP connections
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Cannot listen on %v (%v)", addr, err)
		return err
	}

	var clientID int64
	clientID = 1

	// @TODO maintain a fixed-size connection pool, throw immediate 554s otherwise
	// see http://www.greenend.org.uk/rjk/tech/smtpreplies.html
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Could not handle request:", err)
			continue
		}
		go s.handleSMTP(&Conn{conn})
		clientID++
	}

	return nil

}

func (s *Server) handleMessage(m *email.Message) error {
	fmt.Println(m)
	return nil
}

func (s *Server) handleSMTP(conn *Conn) error {
	defer conn.Close()
	conn.write("220 %v %v", s.Name, time.Now().Format(time.RFC1123Z))

	var errors int
	var isTLS bool

ReadLoop:
	for i := 0; i < s.MaxCommands; i++ {

		input, err := conn.read(s.MaxSize)
		if err != nil {
			log.Printf("Read error: %v", err)
			if err == io.EOF {
				// client closed the connection already
				return nil
			}
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				// too slow, timeout
				return nil
			}

			return err
		}

		cmd := strings.ToUpper(input)

		switch {
		case strings.HasPrefix(cmd, "HELO"):
			conn.write("250 %v Hello ", s.Name)
		case strings.HasPrefix(cmd, "EHLO"):
			conn.write("250-%v Hello [127.0.0.1]", s.Name)
			conn.write("250-SIZE %v", s.MaxSize)
			if !isTLS {
				conn.write("250-STARTTLS")
			}
			conn.write("250 HELP")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			if email, err := extractEmail(input); err == nil {
				log.Println("Message from:", email)
			}
			conn.writeOK()
		case strings.HasPrefix(cmd, "RCPT TO:"):
			if email, err := extractEmail(input); err == nil {
				log.Println("Message to:", email)
			}
			conn.write("250 Accepted")
		case strings.HasPrefix(cmd, "RSET"):
			conn.writeOK()
		case strings.HasPrefix(cmd, "DATA"):
			conn.write("354 Enter message, ending with \".\" on a line by itself")

			if data, err := conn.readData(s.MaxSize); err == nil {

				if message, err := email.NewMessage([]byte(data)); err == nil {

					if err := s.handleMessage(message); err == nil {
						conn.write(fmt.Sprintf("250 OK : queued as %v", message.ID()))
					} else {
						conn.write("554 Error: I blame me.")
					}

				} else {
					conn.write(fmt.Sprintf("554 Error: %v", err))
				}

			} else {
				log.Fatalf("DATA read error: %v", err)
			}

		case strings.HasPrefix(cmd, "STARTTLS"):
			conn.write("220 Ready to start TLS")

			// upgrade to TLS
			tlsConn := tls.Server(conn, TLSConfig)
			err := tlsConn.Handshake()
			if err == nil {
				conn, isTLS = &Conn{tlsConn}, true
			} else {
				log.Fatalf("Could not TLS handshake:%v", err)
			}
		case strings.HasPrefix(cmd, "QUIT"):
			conn.write("221 Bye")
			break ReadLoop
		case strings.HasPrefix(cmd, "NOOP") || strings.HasPrefix(cmd, "XCLIENT"):
			conn.writeOK()
		default:
			conn.write("500 unrecognized command")
			errors++
			if errors > 3 {
				conn.write("500 Too many unrecognized commands")
				break ReadLoop
			}
		}
	}

	return nil
}

func extractEmail(str string) (address string, err error) {
	var host, name string
	re, _ := regexp.Compile(`<(.+?)@(.+?)>`) // go home regex, you're drunk!
	if matched := re.FindStringSubmatch(str); len(matched) > 2 {
		host = validHost(matched[2])
		name = matched[1]
	} else {
		if res := strings.Split(str, "@"); len(res) > 1 {
			name = res[0]
			host = validHost(res[1])
		}
	}
	if host == "" || name == "" {
		err = fmt.Errorf("Invalid address, [%v@%v] address: %v", name, host, str)
	}
	return fmt.Sprintf("%v@%v", name, host), err
}

func validHost(host string) string {
	host = strings.Trim(host, " ")
	re, _ := regexp.Compile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])$`)
	if re.MatchString(host) {
		return host
	}
	return ""
}
