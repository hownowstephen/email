package imap

import (
    "bufio"
    "fmt"
    "net"
    "strings"
    "time"
)

type IMAPClient struct {
    conn net.Conn
}

func NewClient(host string) (*IMAPClient, error) {

    conn, err := net.Dial("tcp", host)
    if err != nil {
        return nil, err
    }

    return &IMAPClient{conn}, nil
}

func (i *IMAPClient) Write(data []byte) (int, error) {
    i.conn.SetDeadline(time.Now().Add(10 * time.Second))
    return i.conn.Write(append(data, []byte("\r\n")...))
}

func (i *IMAPClient) Read() (input string, err error) {
    var reply string
    reader := bufio.NewReader(i.conn)
    for err == nil {
        i.conn.SetDeadline(time.Now().Add(10 * time.Second))
        reply, err = reader.ReadString('\n')
        if reply != "" {
            input = input + reply
        }
        if err != nil {
            break
        }
        if strings.HasSuffix(input, "\r\n") {
            break
        }
    }
    return strings.Trim(input, " \n\r"), err
}

func (i *IMAPClient) Capability() {
    i.Write([]byte("CAPABILITY"))
    response, err := i.Read()
    fmt.Println(response, err)
}
