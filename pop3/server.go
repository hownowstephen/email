package pop3

import (
    "crypto/rand"
    "crypto/tls"
    "fmt"
    "io"
    "log"
    "net"
    "net/mail"
    "os"
    "strings"

    "github.com/hownowstephen/email"
)

const (
    AUTHORIZATION = iota
    TRANSACTION
    UPDATE
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
    RateLimiter func(*Conn) bool

    // Handler is the handoff function for messages
    Handler MessageHandler

    // Maildrop is a dropping point for a mail message
    Maildrop Maildrop

    // // Extensions is a map of server-specific extensions & overrides, by verb
    // Extensions map[string]Extension

    // Disabled features
    Disabled map[string]bool

    // Server flags
    listeners []net.Listener

    // help message to display in response to a HELP request
    Help string
}

// NewServer creates a server with the default settings
func NewServer(maildrop Maildrop) *Server {
    name, err := os.Hostname()
    if err != nil {
        name = "localhost"
    }
    return &Server{
        Name:        name,
        ServerName:  name,
        MaxSize:     131072,
        MaxCommands: 100,
        Maildrop:    maildrop,
        // Extensions:  make(map[string]Extension),
        Disabled: make(map[string]bool),
    }
}

// Close the server connection (not happy with this)
func (s *Server) Close() {
    for _, listener := range s.listeners {
        listener.Close()
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

// SetHelp sets a help message
func (s *Server) SetHelp(message string) error {
    if len(message) > 100 || strings.TrimSpace(message) == "" {
        return fmt.Errorf("Message '%v' is not a valid HELP message. Must be less than 100 characters and non-empty", message)
    }
    s.Help = message
    return nil
}

// ListenAndServe starts listening for SMTP commands at the supplied TCP address
func (s *Server) ListenAndServe(addr string) error {

    // Start listening for POP3 connections
    listener, err := net.Listen("tcp", addr)
    if err != nil {
        log.Fatalf("Cannot listen on %v (%v)", addr, err)
        return err
    }

    var clientID int64
    clientID = 1

    s.listeners = append(s.listeners, listener)

    fmt.Println("LISTENERS", s.listeners)

    // @TODO maintain a fixed-size connection pool, throw immediate 554s otherwise
    // see http://www.greenend.org.uk/rjk/tech/smtpreplies.html
    // https://blog.golang.org/context?
    for {

        conn, err := listener.Accept()

        if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
            // it was a timeout
            continue
        } else if ok && !netErr.Temporary() {
            return netErr
        }

        if err != nil {
            log.Println("Could not handle request:", err)
            continue
        }
        go s.HandlePOP3(&Conn{
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

func (s *Server) HandlePOP3(conn *Conn) error {
    conn.WriteOK("POP3 server ready")

ReadLoop:
    for i := 0; i < s.MaxCommands; i++ {

        var cmd, args string
        var err error

        if cmd, args, err = conn.ReadPOP(); err != nil {
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
        if s.Disabled[cmd] {
            conn.WriteERR("Feature is disabled")
            continue
        }

        // If are in the AUTHORIZATION state (do not have an ongoing Maildrop TRANSACTION)
        // only allow commands specific to that state
        if conn.Maildrop == nil {
            switch cmd {
            case "QUIT", "APOP", "USER", "PASS":
                // these are okay to call in AUTHORIZATION
            default:
                conn.WriteERR("Authentication required")
                continue
            }
        }

        switch cmd {
        case "APOP":

        case "USER":

        case "PASS":

        case "NOOP":
            conn.WriteOK("")
        case "QUIT":
            conn.WriteOK(fmt.Sprintf("%v POP3 server signing off", s.ServerName))
            break ReadLoop
        default:
            conn.WriteERR("Command not understood")
            fmt.Println(cmd, args, err)
        }
    }

    return conn.Close()
}

func (s *Server) GetAddressArg(argName string, args string) (*mail.Address, error) {
    argSplit := strings.SplitN(args, ":", 2)
    if len(argSplit) == 2 && strings.ToUpper(argSplit[0]) == argName {
        return mail.ParseAddress(argSplit[1])
    }

    return nil, fmt.Errorf("Bad arguments")
}
