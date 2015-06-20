package email

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

// type Message interface {
// 	To() []*mail.Address
// 	From() *mail.Address

// 	Headers() map[string]string
// 	Subject() string
// 	Parts() []string
// 	Part(name string) *multipart.Part
// }

// Message is a nicely packaged representation of the
// recieved message
type Message struct {
	To      []*mail.Address
	From    *mail.Address
	Headers map[string]string
	Subject string
	Body    []*Part
	RawBody []byte
}

// Part represents a single part of the message
type Part struct {
	part *multipart.Part
	Body []byte
}

func (m *Message) ID() string {
	return "not-implemented"
}

// Plain returns the text/plain content of the message, if any
func (m *Message) Plain() ([]byte, error) {
	return m.FindByType("text/plain")
}

// HTML returns the text/html content of the message, if any
func (m *Message) HTML() ([]byte, error) {
	return m.FindByType("text/html")
}

// FindByType finds the first part of the message with the specified Content-Type
func (m *Message) FindByType(contentType string) ([]byte, error) {
	for _, p := range m.Body {
		mediaType, _, err := mime.ParseMediaType(p.part.Header.Get("Content-Type"))
		if err == nil && mediaType == contentType {
			return p.Body, nil
		}
	}

	return []byte{}, fmt.Errorf("No %v content found", contentType)
}

// parseBody unwraps the body io.Reader into a set of *Part structs
func parseBody(m *mail.Message) ([]byte, []*Part, error) {

	mbody, err := ioutil.ReadAll(m.Body)
	if err != nil {
		return []byte{}, []*Part{}, err
	}
	buf := bytes.NewBuffer(mbody)

	var parts []*Part

	mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		return mbody, parts, fmt.Errorf("Media Type error: %v", err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(buf, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				return mbody, parts, fmt.Errorf("MIME error: %v", err)
			}

			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				log.Fatal(err)
			}

			parts = append(parts, &Part{p, slurp})

		}
	}
	return mbody, parts, nil
}

// NewMessage creates a Message from a data blob
func NewMessage(data []byte) (*Message, error) {
	m, err := mail.ReadMessage(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	to, err := m.Header.AddressList("to")
	if err != nil {
		return nil, err
	}

	from, err := m.Header.AddressList("from")
	if err != nil {
		return nil, err
	}

	header := make(map[string]string)

	for k, v := range m.Header {
		if len(v) == 1 {
			header[k] = v[0]
		}
	}

	raw, parts, err := parseBody(m)
	if err != nil {
		return nil, err
	}

	return &Message{
		to,
		from[0],
		header,
		m.Header.Get("subject"),
		parts,
		raw,
	}, nil

}
