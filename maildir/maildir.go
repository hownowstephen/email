package maildir

// a simple implementation of http://cr.yp.to/proto/maildir.html

import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "io/ioutil"
    "os"
    "path"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/hownowstephen/email"
)

// MessageHandler is a function for iterating through a maildir
type MessageHandler func(*email.Message) error

// Dir is a single directory containing maildir files
type Dir struct {
    dir      string
    counter  int
    pid      int
    hostname string
}

func exists(p string) bool {
    _, err := os.Stat(p)
    return err == nil
}

func NewDir(dir string) (*Dir, error) {
    // @TODO: maybe check if it's a subdirectory?

    base := path.Dir(dir)
    if !exists(base) {
        if err := os.MkdirAll(base, 0755); err != nil {
            return nil, err
        }
    }

    for _, d := range []string{filepath.Join(base, "tmp"), filepath.Join(base, "cur"), filepath.Join(base, "new")} {
        if !exists(d) {
            if err := os.Mkdir(d, 0755); err != nil {
                return nil, err
            }
        }
    }

    hostname, err := os.Hostname()
    if err != nil {
        return nil, err
    }

    // @TODO: Replace / with \057 and : with \072.

    return &Dir{base, 0, os.Getpid(), hostname}, nil
}

func (d *Dir) Write(m *email.Message) (string, error) {

    filename := d.makeID()

    tmpname := path.Join(d.dir, "tmp", filename)
    f, err := os.Create(tmpname)
    if err != nil {
        return "", err
    }

    // this will be in a weird order. is that a problem?
    for k, v := range m.Headers {
        f.Write([]byte(fmt.Sprintf("%v: %v\n", k, v)))
    }

    f.Write([]byte("\n"))
    f.Write(m.RawBody)
    f.Close()

    return filename, os.Rename(tmpname, filepath.Join(d.dir, "new", filename))
}

// Open a single message from the dir
func (d *Dir) Open(filename string) (*email.Message, error) {

    var f *os.File
    var err error

    f, err = os.Open(path.Join(d.dir, "new", filename))
    if err != nil {
        matches, merr := filepath.Glob(filepath.Join(d.dir, "cur", filename) + "*")
        if merr != nil {
            return nil, merr
        }

        if len(matches) == 1 {
            f, err = os.Open(matches[0])
        } else {
            return nil, fmt.Errorf("Too many matched files: %v", matches)
        }
    }

    if err != nil {
        return nil, err
    }

    defer f.Close()

    b, err := ioutil.ReadAll(f)
    if err != nil {
        return nil, err
    }

    return email.NewMessage(b)
}

func (d *Dir) EachMessage(handler MessageHandler) error {
    return filepath.Walk(d.dir, func(path string, info os.FileInfo, err error) error {

        m, err := d.Open(info.Name())
        if err != nil {
            return err
        }

        return handler(m)
    })
}

func (d *Dir) makeID() string {
    buf := make([]byte, 16)
    rand.Reader.Read(buf)
    d.counter++

    uniq := strings.Join([]string{
        "R", hex.EncodeToString(buf),
        "P", strconv.Itoa(d.pid),
        "Q", strconv.Itoa(d.counter),
    }, "")

    return strings.Join([]string{strconv.Itoa(int(time.Now().Unix())), uniq, d.hostname}, ".")
}
