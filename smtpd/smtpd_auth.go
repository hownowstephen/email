package smtpd

import (
    "fmt"
    "strings"
)

type Auth struct {
    Mechanisms map[string]AuthExtension
}

func NewAuth() *Auth {
    return &Auth{
        Mechanisms: make(map[string]AuthExtension),
    }
}

// Handle authentication by handing off to one of the configured auth mechanisms
func (a *Auth) Handle(c *SMTPConn, args string) error {

    mech := strings.SplitN(args, " ", 2)

    if m, ok := a.Mechanisms[mech[0]]; ok {
        if user, err := m.Handle(c, mech[1]); err == nil {
            c.User = user
            return nil
        } else {
            return err
        }
    }

    return &SMTPError{500, fmt.Errorf("AUTH mechanism %v not available", mech[0])}

}

// EHLO returns a stringified list of the installed Auth mechanisms
func (a *Auth) EHLO() string {
    var mechanisms []string
    for m := range a.Mechanisms {
        mechanisms = append(mechanisms, m)
    }
    return strings.Join(mechanisms, " ")
}

// Extend the auth handler by adding a new mechanism
func (a *Auth) Extend(mechanism string, extension AuthExtension) error {
    if _, ok := a.Mechanisms[mechanism]; ok {
        return fmt.Errorf("AUTH mechanism %v is already implemented", mechanism)
    }
    a.Mechanisms[mechanism] = extension
    return nil
}

// AuthUser should check if a given string identifies that user
type AuthUser interface {
    IsUser(value string) bool
}

// http://tools.ietf.org/html/rfc4422#section-3.1
// https://en.wikipedia.org/wiki/Simple_Authentication_and_Security_Layer
type AuthExtension interface {
    Handle(*SMTPConn, string) (AuthUser, error)
}

type SimpleAuthFunc func(string) (AuthUser, bool)

type AuthPlain struct {
    Auth SimpleAuthFunc
}

// Handles the negotiation of an AUTH PLAIN request
func (a *AuthPlain) Handle(conn *SMTPConn, params string) (AuthUser, error) {

    if !conn.IsTLS {
        return nil, ErrRequiresTLS
    }

    if strings.TrimSpace(params) == "" {
        conn.WriteSMTP(334, "")
        if line, err := conn.ReadUntil("\r\n"); err == nil {
            if user, isAuth := a.Auth(line); isAuth {
                return user, nil
            } else {
                return user, ErrAuthFailed
            }
        } else {
            return nil, err
        }
    } else if user, isAuth := a.Auth(params); isAuth {
        return user, nil
    } else {
        return user, ErrAuthFailed
    }

    return nil, ErrAuthFailed
}
