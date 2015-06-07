package maildir

import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "os"
    "path"
    "strconv"
    "strings"
    "time"

    "github.com/hownowstephen/email"
)

// Dir is a single directory containing maildir files
type Dir struct {
    dir      string
    counter  int
    pid      int
    hostname string
}

func exists(path string) bool {
    _, err := os.Stat(path)
    return !os.IsExist(err)
}

func NewDir(dir string) (*Dir, error) {
    // @TODO: maybe check if it's a subdirectory?

    base := path.Dir(dir)
    if !exists(base) {
        if err := os.Mkdir(base, 0644); err != nil {
            return nil, err
        }
    }

    for _, d := range []string{path.Join(base, "tmp"), path.Join(base, "cur"), path.Join(base, "new")} {
        if !exists(base) {
            if err := os.Mkdir(d, 0644); err != nil {
                return nil, err
            }
        }
    }

    hostname, err := os.Hostname()
    if err != nil {
        return nil, err
    }

    return &Dir{base, 0, os.Getpid(), hostname}, nil
}

func (d *Dir) Write(m *email.Message) error {

    filename := d.makeID()

    tmpname := path.Join(d.dir, "tmp", filename)
    f, err := os.Create(tmpname)
    if err != nil {
        return err
    }

    // this will be in a weird order. is that a problem?
    for k, v := range m.Headers {
        f.Write([]byte(fmt.Sprintf("%v: %v\n", k, v)))
    }

    f.Write([]byte("\n"))
    f.Write(m.RawBody)
    f.Close()

    return os.Rename(tmpname, path.Join(d.dir, "new", filename))
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
