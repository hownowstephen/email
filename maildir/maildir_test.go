package maildir

import (
    "fmt"
    "os"
    "path/filepath"
    "reflect"
    "testing"

    "github.com/hownowstephen/email"
)

func TestCreate(t *testing.T) {

    defer os.RemoveAll("tmp")

    if err := os.MkdirAll("tmp/maildir-test", 0755); err != nil {
        t.Errorf("Couldn't create testing dir: %v", err)
    }

    _, err := NewDir("tmp/maildir-test/")
    if err != nil {
        t.Errorf("Couldn't create a maildir: %v", err)
    }

    if !exists("tmp/maildir-test/tmp") {
        t.Errorf("Populating children of maildir failed.")
    }

}

func TestMakeID(t *testing.T) {

    defer os.RemoveAll("tmp")

    if err := os.MkdirAll("tmp/maildir-test", 0755); err != nil {
        t.Errorf("Couldn't create testing dir: %v", err)
    }

    dir, err := NewDir("tmp/maildir-test/")
    if err != nil {
        t.Errorf("Couldn't create a maildir: %v", err)
    }

    id1 := dir.makeID()
    c1 := dir.counter

    id2 := dir.makeID()
    c2 := dir.counter

    if id1 == id2 {
        t.Errorf("Ids should be uniquely generated, got %v twice", id1)
    }

    if c2 <= c1 {
        t.Errorf("Counter value should be increasing, got: %v after %v", c2, c1)
    }

}

func TestWriteMessage(t *testing.T) {
    defer os.RemoveAll("tmp")

    dir, err := NewDir("tmp/maildir-test/")
    if err != nil {
        t.Errorf("Couldn't create a maildir: %v", err)
    }

    rawMessage := `To: sender@example.org
From: recipient@example.net
Content-Type: text/html

This is the email body`

    m, err := email.NewMessage([]byte(rawMessage))

    if err != nil {
        t.Errorf("Example message unparseable: %v", err)
    }

    filename, err := dir.Write(m)

    if err != nil {
        t.Errorf("Couldn't write message to maildir: %v", err)
    }

    info, err := os.Stat(fmt.Sprintf("tmp/maildir-test/new/%v", filename))

    if err != nil {
        t.Errorf("Couldn't stat '%v': %v", filename, err)
    }

    if int(info.Size()) != len(rawMessage) {
        t.Errorf("Size of file doesn't match size of message? want: %v, got: %v", len(rawMessage), info.Size())
    }

}

func TestOpen(t *testing.T) {
    defer os.RemoveAll("tmp")

    dir, err := NewDir("tmp/maildir-test/")
    if err != nil {
        t.Errorf("Couldn't create a maildir: %v", err)
    }

    rawMessage := `To: sender@example.org
From: recipient@example.net
Content-Type: text/html

This is the email body`

    m, err := email.NewMessage([]byte(rawMessage))

    if err != nil {
        t.Errorf("Example message unparseable: %v", err)
    }

    filename, err := dir.Write(m)

    n, err := dir.Open(filename)
    if err != nil {
        t.Errorf("Couldn't open message: %v", err)
    }

    if !reflect.DeepEqual(m, n) {
        t.Errorf("Reloaded message doesn't match saved message! want: %v, got: %v", m, n)
    }

    err = os.Rename(filepath.Join(dir.dir, "new", filename), filepath.Join(dir.dir, "cur", filename)+":PRS")
    if err != nil {
        t.Errorf("Couldn't move file to cur: %v", err)
    }

    c, err := dir.Open(filename)
    if err != nil {
        t.Errorf("Couldn't open message from cur: %v", err)
    }

    if !reflect.DeepEqual(m, c) {
        t.Errorf("Message from cur didn't match original! want: %v, got: %v", m, c)
    }

}
