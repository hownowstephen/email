package smtpd

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type Conn struct {
	net.Conn
	IsTLS   bool
	Errors  []error
	MaxSize int
}

// ReadSMTP
func (c *Conn) ReadSMTP() (string, error) {
	return c.ReadUntil("\r\n")
}

// readData brokers the special case of SMTP data messages
func (c *Conn) ReadData() (string, error) {
	return c.ReadUntil("\r\n.\r\n")
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

// WriteSMTP writes a general SMTP line
func (c *Conn) WriteSMTP(code int, message string) error {
	log.Println(code, message)
	c.SetDeadline(time.Now().Add(10 * time.Second))
	_, err := c.Write([]byte(fmt.Sprintf("%v %v", code, message) + "\r\n"))
	return err
}

// WriteEHLO writes an EHLO line, see https://tools.ietf.org/html/rfc2821#section-4.1.1.1
func (c *Conn) WriteEHLO(message string) error {
	log.Println("EHLO", message)
	c.SetDeadline(time.Now().Add(10 * time.Second))
	_, err := c.Write([]byte(fmt.Sprintf("250-%v", message) + "\r\n"))
	return err
}

// WriteOK is a convenience function for sending the default OK response
func (c *Conn) WriteOK() error {
	return c.WriteSMTP(250, "OK")
}
