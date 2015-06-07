package maildir

import (
    "os"
    "testing"
)

func TestCreate(t *testing.T) {

    defer os.RemoveAll("tmp")

    if err := os.MkdirAll("tmp/maildir1", 0755); err != nil {
        t.Errorf("Couldn't create testing dir: %v", err)
    }

    _, err := NewDir("tmp/maildir1/")
    if err != nil {
        t.Errorf("Couldn't create a maildir: %v", err)
    }

    if !exists("tmp/maildir1/tmp") {
        t.Errorf("Populating children of maildir failed.")
    }

}

func TestMakeID(t *testing.T) {

    defer os.RemoveAll("tmp")

    if err := os.MkdirAll("tmp/maildir1", 0755); err != nil {
        t.Errorf("Couldn't create testing dir: %v", err)
    }

    dir, err := NewDir("tmp/maildir1/")
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
