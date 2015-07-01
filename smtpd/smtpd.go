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

// MessageHandler functions handle application of business logic to the inbound message
type MessageHandler func(m *email.Message) error

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
    RateLimiter func(*SMTPConn) bool

    // Handler is the handoff function for messages
    Handler MessageHandler

    // Auth is an authentication-handling extension
    Auth Extension

    // Extensions is a map of server-specific extensions & overrides, by verb
    Extensions map[string]Extension

    // Disabled features
    Disabled map[string]bool

    // Server flags
    listeners []net.Listener
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
        Extensions:  make(map[string]Extension),
        Disabled:    make(map[string]bool),
    }
}

// Close the server connection (not happy with this)
func (s *Server) Close() {
    for _, listener := range s.listeners {
        listener.Close()
    }
}

func (s *Server) Greeting(conn *SMTPConn) string {
    return fmt.Sprintf("Welcome! [%v]", conn.LocalAddr())
}

func (s *Server) Extend(verb string, extension Extension) error {
    if _, ok := s.Extensions[verb]; ok {
        return fmt.Errorf("Extension for %v has already been registered", verb)
    }

    s.Extensions[verb] = extension
    return nil
}

// Disable server capabilities
func (s *Server) Disable(verbs ...string) {
    for _, verb := range verbs {
        s.Disabled[strings.ToUpper(verb)] = true
    }
}

// Enable server capabilities that have previously been disabled
func (s *Server) Enable(verbs ...string) {
    for _, verb := range verbs {
        s.Disabled[strings.ToUpper(verb)] = false
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
        ServerName:   s.ServerName,
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

    // var listener *net.TCPListener
    // if tl, ok := l.(*net.TCPListener); ok {
    //     listener = tl
    // } else {
    //     log.Fatalf("Couldn't open a TCP listener, got %v instead", l)
    // }

    var clientID int64
    clientID = 1

    s.listeners = append(s.listeners, listener)

    // @TODO maintain a fixed-size connection pool, throw immediate 554s otherwise
    // see http://www.greenend.org.uk/rjk/tech/smtpreplies.html
    // https://blog.golang.org/context?
    for {

        conn, err := listener.Accept()

        if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
            // it was a timeout
            continue
        } else if ok && !netErr.Temporary() {
            break
        }

        if err != nil {
            log.Println("Could not handle request:", err)
            continue
        }
        go s.HandleSMTP(&SMTPConn{
            Conn:    conn,
            IsTLS:   false,
            Errors:  []error{},
            MaxSize: s.MaxSize,
        })
        clientID++

    }
    return nil

}

func (s *Server) Address() string {
    if len(s.listeners) > 0 {
        return s.listeners[0].Addr().String()
    }
    return ""
}

func (s *Server) handleMessage(m *email.Message) error {
    return s.Handler(m)
}

func (s *Server) HandleSMTP(conn *SMTPConn) error {
    defer conn.Close()
    conn.WriteSMTP(220, fmt.Sprintf("%v %v", s.Name, time.Now().Format(time.RFC1123Z)))

ReadLoop:
    for i := 0; i < s.MaxCommands; i++ {

        var verb, args string
        var err error

        if verb, args, err = conn.ReadSMTP(); err != nil {
            log.Printf("Read error: %v", err)
            if err == io.EOF {
                // client closed the connection already
                break ReadLoop
            }
            if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
                // too slow, timeout
                break ReadLoop
            }

            return err
        }

        // Always check for disabled features first
        if s.Disabled[verb] {
            if verb == "EHLO" {
                conn.WriteSMTP(550, "Not implemented")
            } else {
                conn.WriteSMTP(502, "Command not implemented")
            }
            continue
        }

        // Handle any extensions / overrides before running default logic
        if _, ok := s.Extensions[verb]; ok {
            err := s.Extensions[verb].Handle(conn, args)
            if err != nil {
                log.Printf("Error? %v", err)
            }
            continue
        }

        switch verb {
        // https://tools.ietf.org/html/rfc2821#section-4.1.1.1
        case "HELO":
            conn.WriteSMTP(250, fmt.Sprintf("%v Hello", s.Name))
        case "EHLO":
            conn.WriteEHLO(fmt.Sprintf("%v %v", s.ServerName, s.Greeting(conn)))
            conn.WriteEHLO(fmt.Sprintf("SIZE %v", s.MaxSize))
            if !conn.IsTLS {
                conn.WriteEHLO("STARTTLS")
            }
            if !conn.IsAuthenticated && s.Auth != nil {
                conn.WriteEHLO(fmt.Sprintf("AUTH %v", s.Auth.EHLO()))
            }
            for verb, extension := range s.Extensions {
                conn.WriteEHLO(fmt.Sprintf("%v %v", verb, extension.EHLO()))
            }
            conn.WriteSMTP(250, "HELP")
        // https://tools.ietf.org/html/rfc2821#section-4.1.1.2
        // see also: http://tools.ietf.org/html/rfc4954#section-3
        //  5.  An optional parameter using the keyword "AUTH" is added to the
        // MAIL FROM command, and extends the maximum line length of the
        // MAIL FROM command by 500 characters.
        case "MAIL":
            // This is wrong, won't always be an email address
            if email, err := extractEmail("to", args); err == nil {
                log.Println("Message from:", email)
            }
            conn.WriteOK()
        // https://tools.ietf.org/html/rfc2821#section-4.1.1.3
        case "RCPT":
            if email, err := extractEmail("from", args); err == nil {
                log.Println("Message to:", email)
            }
            conn.WriteSMTP(250, "Accepted")
        // https://tools.ietf.org/html/rfc2821#section-4.1.1.4
        case "DATA":
            conn.WriteSMTP(354, "Enter message, ending with \".\" on a line by itself")

            if data, err := conn.ReadData(); err == nil {

                if message, err := email.NewMessage([]byte(data)); err == nil {

                    if err := s.handleMessage(message); err == nil {
                        conn.WriteSMTP(250, fmt.Sprintf("OK : queued as %v", message.ID()))
                    } else {
                        conn.WriteSMTP(554, fmt.Sprintf("Error: I blame me. %v", err))
                    }

                } else {
                    conn.WriteSMTP(554, fmt.Sprintf("Error: I blame you. %v", err))
                }

            } else {
                log.Fatalf("DATA read error: %v", err)
            }
        // https://tools.ietf.org/html/rfc2821#section-4.1.1.5
        case "RSET":
            conn.WriteOK()
            return s.HandleSMTP(conn)

        // https://tools.ietf.org/html/rfc2821#section-4.1.1.6
        case "VRFY":
            conn.WriteOK()

        // https://tools.ietf.org/html/rfc2821#section-4.1.1.7
        case "EXPN":
            conn.WriteOK()

        // https://tools.ietf.org/html/rfc2821#section-4.1.1.8
        case "HELP":
            conn.WriteOK()

        // https://tools.ietf.org/html/rfc2821#section-4.1.1.9
        case "NOOP":
            conn.WriteOK()

        // https://tools.ietf.org/html/rfc2821#section-4.1.1.10
        case "QUIT":
            conn.WriteSMTP(221, "Bye")
            break ReadLoop

        // https://tools.ietf.org/html/rfc2487
        case "STARTTLS":
            conn.WriteSMTP(220, "Ready to start TLS")

            // upgrade to TLS
            tlsConn := tls.Server(conn, s.TLSConfig)
            if tlsConn == nil {
                log.Fatalf("Couldn't upgrade to TLS")
            }
            if err := tlsConn.Handshake(); err == nil {
                conn = &SMTPConn{
                    Conn:            tlsConn,
                    IsTLS:           true,
                    IsAuthenticated: conn.IsAuthenticated,
                    Errors:          conn.Errors,
                    MaxSize:         conn.MaxSize,
                }
            } else {
                log.Fatalf("Could not TLS handshake:%v", err)
            }

        case "AUTH":
            if conn.IsAuthenticated {
                conn.WriteSMTP(503, "You are already authenticated")
            } else if s.Auth != nil {
                if err := s.Auth.Handle(conn, args); err != nil {
                    if authErr, ok := err.(*AuthError); ok {
                        conn.WriteSMTP(authErr.Code(), authErr.Error())
                    } else {
                        conn.WriteSMTP(500, "Authentication failed")
                    }
                } else {
                    conn.WriteSMTP(235, "Authentication succeeded")
                }
            } else {
                conn.WriteSMTP(502, "Command not implemented")
            }
        default:

            conn.WriteSMTP(500, "Syntax error, command unrecognised")
            conn.Errors = append(conn.Errors, fmt.Errorf("bad input: %v %v", verb, args))
            if len(conn.Errors) > 3 {
                conn.WriteSMTP(500, "Too many unrecognized commands")
                break ReadLoop
            }

        }
    }

    return nil
}

func extractEmail(param, str string) (address string, err error) {
    var host, name string
    re, _ := regexp.Compile(fmt.Sprintf(`(?i)%v:<(.+?)@(.+?)>`, param))
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
