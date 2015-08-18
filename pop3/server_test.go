package pop3_test

import (
    "testing"
    "time"

    "github.com/hownowstephen/email/pop3"
)

func TestPOP3Server(t *testing.T) {

    server := pop3.NewServer(&TestMaildrop{})
    go server.ListenAndServe(":0")
    time.Sleep(time.Second)

    client, err := pop3.Dial(server.Address())
    if err != nil {
        t.Errorf("Couldn't dial the server! %v", err)
    }

    client.Quit()
}
