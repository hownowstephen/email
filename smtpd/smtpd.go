package smtpd

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/hownowstephen/email"
)

// MessageHandler functions handle application of business logic to the inbound message
type MessageHandler func(m *email.Message) (string, error)

// X509 certificate path, see http://www.ipsec-howto.org/x595.html
const (
	X509PUB  = "../certs/server.crt"
	X509PRIV = "../certs/server.key"
)

// @TODO Refactor these into the underlying mail handler
const (
	SERVERNAME = "mail.hownowstephen.com"
	MAXSIZE    = 131072
)

// TLSConfig handles certificates & handshaking, if available
var TLSConfig *tls.Config

// ListenAndServe creates a Server with a very general set of options
func ListenAndServe(addr string, handler MessageHandler) error {
	if cert, err := tls.LoadX509KeyPair(X509PUB, X509PRIV); err == nil {
		TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.VerifyClientCertIfGiven,
			Rand:         rand.Reader,
		}
	} else {
		fmt.Println("Could not load TLS keypair, %v", err)
	}

	// Start listening for SMTP connections
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Cannot listen on %v (%v)", addr, err)
	}

	log.Println("Listen...")

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
		go handleSMTP(conn, handler)
		clientID++
	}

}

func handleSMTP(conn Conn, handler func(m *email.Message) (string, error)) {
	defer conn.Close()
	write(conn, "220 %v %v", SERVERNAME, time.Now().Format(time.RFC1123Z))

	var errors int
	var isTLS bool

ReadLoop:
	for i := 0; i < 100; i++ {

		input, err := read(conn)
		if err != nil {
			log.Printf("Read error: %v", err)
			if err == io.EOF {
				// client closed the connection already
				return
			}
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				// too slow, timeout
				return
			}
			break
		}

		cmd := strings.ToUpper(input)

		switch {
		case strings.HasPrefix(cmd, "HELO"):
			write(conn, "250 %v Hello ", SERVERNAME)
		case strings.HasPrefix(cmd, "EHLO"):
			write(conn, "250-%v Hello [127.0.0.1]", SERVERNAME)
			write(conn, "250-SIZE %v", MAXSIZE)
			if !isTLS {
				write(conn, "250-STARTTLS")
			}
			write(conn, "250 HELP")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			if email, err := extractEmail(input); err == nil {
				log.Println("Message from:", email)
			}
			writeOK(conn)
		case strings.HasPrefix(cmd, "RCPT TO:"):
			if email, err := extractEmail(input); err == nil {
				log.Println("Message to:", email)
			}
			write(conn, "250 Accepted")
		case strings.HasPrefix(cmd, "RSET"):
			writeOK(conn)
		case strings.HasPrefix(cmd, "DATA"):
			write(conn, "354 Enter message, ending with \".\" on a line by itself")

			if data, err := readData(conn); err == nil {

				if message, err := email.NewMessage(data); err == nil {

					if id, err := handler(message); err == nil {
						write(conn, fmt.Sprintf("250 OK : queued as %v", id))
					} else {
						write(conn, "554 Error: I blame me.")
					}

				} else {
					write(conn, fmt.Sprintf("554 Error: %v", err))
				}

			} else {
				log.Fatalf("DATA read error: %v", err)
			}

		case strings.HasPrefix(cmd, "STARTTLS"):
			write(conn, "220 Ready to start TLS")

			// upgrade to TLS
			tlsConn := tls.Server(conn, TLSConfig)
			err := tlsConn.Handshake()
			if err == nil {
				conn, isTLS = tlsConn, true
			} else {
				log.Fatalf("Could not TLS handshake:%v", err)
			}
		case strings.HasPrefix(cmd, "QUIT"):
			write(conn, "221 Bye")
			break ReadLoop
		case strings.HasPrefix(cmd, "NOOP") || strings.HasPrefix(cmd, "XCLIENT"):
			writeOK(conn)
		default:
			write(conn, "500 unrecognized command")
			errors++
			if errors > 3 {
				write(conn, "500 Too many unrecognized commands")
				break ReadLoop
			}
		}
	}
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
