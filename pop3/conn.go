package pop3

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "strings"
    "sync"
    "time"
)

type Conn struct {
    net.Conn
    IsTLS    bool
    Errors   []error
    MaxSize  int
    Maildrop Maildrop
    lock     sync.Mutex
    // transaction int
    State int
}

func (c *Conn) Reset() {
}

// ReadSMTP pulls a single POP3 command line (ending in a carriage return + newline (aka CRLF))
func (c *Conn) ReadPOP() (string, string, error) {
    if value, err := c.ReadUntil("\r\n"); err == nil {
        value = strings.TrimSpace(value)

        var args string
        command := strings.SplitN(value, " ", 2)

        verb := strings.ToUpper(command[0])
        if len(command) > 1 {
            args = command[1]
        }

        log.Println("C:", verb, args)
        return verb, args, nil
    } else {
        return "", "", err
    }
}

// rawRead performs the actual read from the connection, reading each line up to the first occurrence of suffix
func (c *Conn) ReadUntil(suffix string) (value string, err error) {
    var reply string
    reader := bufio.NewReader(c)
    for err == nil {
        c.SetDeadline(time.Now().Add(10 * time.Second))
        reply, err = reader.ReadString('\n')
        if reply != "" {
            value = value + reply
            if len(value) > c.MaxSize && c.MaxSize > 0 {
                return "", fmt.Errorf("Maximum DATA size exceeded (%v)", c.MaxSize)
            }
        }
        if err != nil {
            break
        }
        if strings.HasSuffix(value, suffix) {
            break
        }
    }
    return value, err
}

// writePOP sends a POP3-formatted response message
func (c *Conn) writePOP(code, message string) error {
    log.Println("S:", code, message)
    c.SetDeadline(time.Now().Add(10 * time.Second))
    _, err := c.Write([]byte(fmt.Sprintf("%v %v", code, message) + "\r\n"))
    return err
}

// WriteOK response with a POP3 success message
func (c *Conn) WriteOK(message string) error {
    return c.writePOP("+OK", message)
}

// WriteERR responds with a POP3 failure message
func (c *Conn) WriteERR(message string) error {
    return c.writePOP("-ERR", message)
}
