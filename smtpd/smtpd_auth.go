package smtpd

import (
    "fmt"
    "strings"
)

type AuthExtension interface {
    Handle(*SMTPConn, string) error
}

type Auth struct {
    Mechanisms map[string]AuthExtension
}

func (a *Auth) Handle(c *SMTPConn, args string) error {

    mech := strings.SplitN(args, " ", 2)

    if m, ok := a.Mechanisms[mech[0]]; ok {
        return m.Handle(c, mech[1])
    } else {
        return fmt.Errorf("AUTH mechanism %v not available", mech[0])
    }
}

func (a *Auth) EHLO() string {
    var mechanisms []string
    for m := range a.Mechanisms {
        mechanisms = append(mechanisms, m)
    }
    return strings.Join(mechanisms, " ")
}

func (a *Auth) Extend(mechanism string, extension AuthExtension) error {
    if _, ok := a.Mechanisms[mechanism]; ok {
        return fmt.Errorf("AUTH mechanism %v is already implemented", mechanism)
    }
    a.Mechanisms[mechanism] = extension
    return nil
}
