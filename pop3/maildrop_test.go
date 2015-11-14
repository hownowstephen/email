package pop3_test

type TestMaildrop struct{}

func (t *TestMaildrop) Count() int {
    return 0
}

func (t *TestMaildrop) Flag(message int) error {
    return nil
}

func (t *TestMaildrop) Delete() error {
    return nil
}

func (t *TestMaildrop) Close() error {
    return nil
}
