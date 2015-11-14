package pop3

type Auth interface {
    Auth(conn *Conn) (Maildrop, error)
}

type Maildrop interface {

    // Count returns the number of messages in the maildrop
    Count() int

    // Size returns the size of the maildrop, in octets
    Size() int

    // Get the message
    Get(message int) Message

    // Messages is a list of messages
    Messages() []Message

    // Flag a message for deletion
    Flag(message int) error

    // Delete performs the UPDATE state deletion step
    Delete() error

    // Close the maildrop
    Close() error
}

type Message interface {
    Id() int
    Size() int
}
