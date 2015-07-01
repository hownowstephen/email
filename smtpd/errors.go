package smtpd

import "errors"

var ErrAuthFailed = &SMTPError{535, errors.New("Authentication credentials invalid")}
var ErrRequiresTLS = &SMTPError{538, errors.New("Encryption required for requested authentication mechanism")}

// SMTPError is an error + SMTP response code
type SMTPError struct {
    code int
    err  error
}

// Code pulls the code
func (a *SMTPError) Code() int {
    return a.code
}

// Error pulls the base error value
func (a *SMTPError) Error() string {
    return a.err.Error()
}
