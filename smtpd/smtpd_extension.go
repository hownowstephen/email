package smtpd

type Extension interface {
    Handle(*SMTPConn, string) error
    EHLO() string
}

type SimpleExtension struct {
    Handler func(*SMTPConn, string) error
    Ehlo    string
}

func (s *SimpleExtension) Handle(c *SMTPConn, args string) error {
    return s.Handler(c, args)
}

func (s *SimpleExtension) EHLO() string {
    return s.Ehlo
}
