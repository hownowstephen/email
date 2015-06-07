package smtpd

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
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

	// RateLimiter gets called before proceeding through to message handling
	RateLimiter func(Conn) bool
}

// UseTLS tries to enable TLS on the server (can also just explicitly set the TLSConfig)
func (s *Server) UseTLS(cert, key string) error {
	c, err := tls.LoadX509KeyPair(X509PUB, X509PRIV)
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
