package imap

import "testing"

func TestClientCapability(t *testing.T) {
    client, err := NewClient("imap.gmail.com:993")
    if err != nil {
        t.Errorf("Couldn't create client: %v", err)
    }

    client.Capability()
}
