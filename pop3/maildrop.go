package pop3

type Auth interface {
    Auth(conn *Conn) (Maildrop, error)
}

type Maildrop interface {

    // Lock the maildrop
    Lock() error

    // Unlock the maildrop
    Unlock() error

    // Count returns the number of messages in the maildrop
    Count() int

    // Flag a message for deletion
    Flag(message int) error

    // Delete performs the UPDATE state deletion step
    Delete() error
}
