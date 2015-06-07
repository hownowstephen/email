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
