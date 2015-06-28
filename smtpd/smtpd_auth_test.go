package smtpd_test

import (
    "crypto/tls"
    "net/smtp"
    "testing"
    "time"

    "github.com/hownowstephen/email/smtpd"
)

func TestSMTPAuthPLAIN(t *testing.T) {

    recorder := &MessageRecorder{}
    server := smtpd.NewServer(recorder.Record)
    if err := server.UseTLS("./server.crt", "./server.key"); err != nil {
        t.Errorf("Server couldn't load TLS credentials")
    }
    go server.ListenAndServe("localhost:12525")
    defer server.Close()

    time.Sleep(time.Second)

    // Connect to the remote SMTP server.
    c, err := smtp.Dial("127.0.0.1:12525")
    if err != nil {
        t.Errorf("Should be able to dial localhost: %v", err)
    }

    if err := c.StartTLS(&tls.Config{ServerName: server.ServerName, InsecureSkipVerify: true}); err != nil {
        t.Errorf("Should be able to negotiate some TLS? %v", err)
    }

    auth := smtp.PlainAuth("", "user@example.com", "password", server.ServerName)

    if err := c.Auth(auth); err != nil {
        t.Errorf("Auth should have succeeded: %v", err)
    }

}

func TestSMTPAuthLocking(t *testing.T) {
    recorder := &MessageRecorder{}
    server := smtpd.NewServer(recorder.Record)
    go server.ListenAndServe("localhost:12525")
    defer server.Close()

    time.Sleep(time.Second)

    // Connect to the remote SMTP server.
    c, err := smtp.Dial("127.0.0.1:12525")
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
    go server.ListenAndServe("localhost:12525")
    defer server.Close()

    time.Sleep(time.Second)

    // Connect to the remote SMTP server.
    c, err := smtp.Dial("127.0.0.1:12525")
    if err != nil {
        t.Errorf("Should be able to dial localhost: %v", err)
    }

    auth := smtp.PlainAuth("", "user@example.com", "password", "mail.example.com")

    if err := c.Auth(auth); err == nil {
        t.Errorf("Should not be able to do PLAIN auth on an unencrypted connection")
    }
}
