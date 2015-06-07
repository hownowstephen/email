package smtpd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

// Message is a nicely packaged representation of the
// recieved message
type Message struct {
	To      []*mail.Address
	From    *mail.Address
	Headers map[string]string
	Subject string
	Body    []*Part
}

// Part represents a single part of the message
type Part struct {
	part *multipart.Part
	Body []byte
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
func parseBody(m *mail.Message) ([]*Part, error) {

	var parts []*Part

	mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		return parts, fmt.Errorf("Media Type error: %v", err)
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(m.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				return parts, fmt.Errorf("MIME error: %v", err)
			}

			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				log.Fatal(err)
			}

			parts = append(parts, &Part{p, slurp})

		}
	}
	return parts, nil
}

// NewMessage creates a Message from a data blob
func NewMessage(data string) (*Message, error) {
	m, err := mail.ReadMessage(strings.NewReader(data))
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
		uc := strings.ToUpper(k)
		if uc == "TO" || uc == "FROM" {
			continue
		} else if len(v) == 1 {
			header[k] = v[0]
		}
	}

	body, err := parseBody(m)
	if err != nil {
		return nil, err
	}

	return &Message{
		to,
		from[0],
		header,
		m.Header.Get("subject"),
		body,
	}, nil

}
