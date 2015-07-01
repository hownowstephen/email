package smtpd_test

import (
    "crypto/tls"
    "encoding/base64"
    "net/smtp"
    "strings"
    "testing"
    "time"

    "github.com/hownowstephen/email/smtpd"
)

type TestUser struct {
    actor    string
    username string
    password string
}

func (t *TestUser) IsUser(ident string) bool {
    return true
}

func TestSMTPAuthPlain(t *testing.T) {
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

func TestSMTPAuthPlainRejection(t *testing.T) {
    recorder := &MessageRecorder{}
    server := smtpd.NewServer(recorder.Record)

    passwd := map[string]string{
        "user@example.com": "password",
        "user@example.ca":  "canadian-password",
    }

    serverAuth := smtpd.NewAuth()
    serverAuth.Extend("PLAIN", &smtpd.AuthPlain{
        Auth: func(value string) (smtpd.AuthUser, bool) {
            rawCreds, err := base64.StdEncoding.DecodeString(value)
            if err != nil {
                return nil, false
            }
            creds := strings.SplitN(string(rawCreds), "\x00", 3)

            if len(creds) != 3 {
                return nil, false
            }

            user := &TestUser{creds[0], creds[1], creds[2]}

            if passwd[user.username] == user.password {
                return user, true
            }

            return nil, false
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
        return
    }

    c.StartTLS(&tls.Config{ServerName: server.Name, InsecureSkipVerify: true})

    auth := smtp.PlainAuth("", "user@example.com", "password", "127.0.0.1")

    if err := c.Auth(auth); err != nil {
        t.Errorf("Auth should have succeded! %v", err)
    }

    // Connect to the remote SMTP server.
    c, err = smtp.Dial(server.Address())
    if err != nil {
        t.Errorf("Should be able to dial localhost: %v", err)
        return
    }

    c.StartTLS(&tls.Config{ServerName: server.Name, InsecureSkipVerify: true})

    auth = smtp.PlainAuth("", "user@example.ca", "password", "127.0.0.1")

    if err := c.Auth(auth); err == nil {
        t.Errorf("Auth should have failed!")
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

func TestSMTPAuthPlainEncryption(t *testing.T) {
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
