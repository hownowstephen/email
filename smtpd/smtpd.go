package smtpd

import (
	"crypto/tls"

	"github.com/hownowstephen/email"
)

// MessageHandler functions handle application of business logic to the inbound message
type MessageHandler func(m *email.Message) error

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
func ListenAndServeSMTP(addr string, handler MessageHandler) error {
	server := NewServer(handler)
	server.UseTLS(X509PUB, X509PRIV)
	return server.ListenAndServe(addr)
}
