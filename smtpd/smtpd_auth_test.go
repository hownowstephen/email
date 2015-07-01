package smtpd_test

import (
    "crypto/tls"
    "net/smtp"
    "testing"
    "time"

    "github.com/hownowstephen/email/smtpd"
)

type TestUser struct{}

func (t *TestUser) IsUser(ident string) bool {
    return true
}

func TestSMTPAuthPLAIN(t *testing.T) {

    recorder := &MessageRecorder{}
    server := smtpd.NewServer(recorder.Record)

    serverAuth := smtpd.NewAuth()
    serverAuth.Extend("PLAIN", &smtpd.AuthPlain{
        Auth: func(value string) (smtpd.AuthUser, bool) {
            return &TestUser{}, true
        },
    })

    server.Auth = serverAuth

    // to generate: http://www.akadia.com/services/ssh_test_certificate.html
    if err := server.UseTLS("./server.crt", "./server.key"); err != nil {
        t.Errorf("Server couldn't load TLS credentials")
    }
    go server.ListenAndServe("localhost:0")
    defer server.Close()

    time.Sleep(time.Second)

    // Connect to the remote SMTP server.
    c, err := smtp.Dial(server.Address())
    if err != nil {
        t.Errorf("Should be able to dial localhost: %v", err)
    }

    if err := c.StartTLS(&tls.Config{ServerName: server.Name, InsecureSkipVerify: true}); err != nil {
        t.Errorf("Should be able to negotiate some TLS? %v", err)
    }

    auth := smtp.PlainAuth("", "user@example.com", "password", "127.0.0.1")

    if err := c.Auth(auth); err != nil {
        t.Errorf("Auth should have succeeded: %v", err)
    }

}

func TestSMTPAuthLocking(t *testing.T) {
    recorder := &MessageRecorder{}
    server := smtpd.NewServer(recorder.Record)

    serverAuth := smtpd.NewAuth()
    serverAuth.Extend("PLAIN", &smtpd.AuthPlain{
        Auth: func(value string) (smtpd.AuthUser, bool) {
            return &TestUser{}, true
        },
    })

    server.Auth = serverAuth

    go server.ListenAndServe("localhost:0")
    defer server.Close()

    time.Sleep(time.Second)

    // Connect to the remote SMTP server.
    c, err := smtp.Dial(server.Address())
    if err != nil {
        t.Errorf("Should be able to dial localhost: %v", err)
    }

    if err := c.Mail("sender@example.org"); err == nil {
        t.Errorf("Should not be able to set a sender before Authenticating")
    }
}

func TestSMTPAuthPLAINEncryption(t *testing.T) {
    recorder := &MessageRecorder{}
    server := smtpd.NewServer(recorder.Record)

    serverAuth := smtpd.NewAuth()
    serverAuth.Extend("PLAIN", &smtpd.AuthPlain{
        Auth: func(value string) (smtpd.AuthUser, bool) {
            return &TestUser{}, true
        },
    })

    server.Auth = serverAuth

    go server.ListenAndServe("localhost:0")
    defer server.Close()

    time.Sleep(time.Second)

    // Connect to the remote SMTP server.
    c, err := smtp.Dial(server.Address())
    if err != nil {
        t.Errorf("Should be able to dial localhost: %v", err)
    }

    auth := smtp.PlainAuth("", "user@example.com", "password", "127.0.0.1")

    if err := c.Auth(auth); err == nil {
        t.Errorf("Should not be able to do PLAIN auth on an unencrypted connection")
    }
}
