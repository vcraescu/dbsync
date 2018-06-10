package watcher

import (
	"time"
	"log"

	"github.com/vcraescu/dbsync/internal/database/mysql"
)

// Watcher - watch for db changes
type Watcher struct {
	conn   *mysql.Connection
	poll   time.Duration
	DiffCh chan Diff
	ErrCh  chan error
}

// Diff - checksums diff
type Diff struct {
	Updated []string
	Created []string
	Deleted []string
}

// New - creates new watcher
func New(conn *mysql.Connection, poll time.Duration) *Watcher {
	w := &Watcher{
		conn:   conn,
		poll:   poll,
		DiffCh: make(chan Diff, 100),
		ErrCh:  make(chan error, 100),
	}

	return w
}

func (w *Watcher) Hostname() string {
	return w.conn.Host
}

func (w *Watcher) DBName() string {
	return w.conn.DBName
}

func newDiff(o map[string]string, n map[string]string) *Diff {
	d := &Diff{
		Updated: make([]string, 0),
		Created: make([]string, 0),
		Deleted: make([]string, 0),
	}

	for name, newChks := range n {
		oldChks, ok := o[name]
		if !ok {
			d.Created = append(d.Created, name)
			continue
		}

		if oldChks != newChks {
			d.Updated = append(d.Updated, name)
		}
	}

	for name := range o {
		_, ok := n[name]
		if !ok {
			d.Deleted = append(d.Deleted, name)
		}
	}

	return d
}

// Empty - returns true when diff is empty
func (d *Diff) Empty() bool {
	return len(d.Updated) == 0 && len(d.Created) == 0 && len(d.Deleted) == 0
}

// Start - starts watching for changes
func (w *Watcher) Start() error {
	err := w.conn.Open()
	if err != nil {
		return err
	}
	defer w.conn.Close()

	var lastChks map[string]string

	for {
		time.Sleep(w.poll)
		log.Println("Checking for changes...")
		chks, err := w.conn.TableChecksums()
		if err != nil {
			w.ErrCh <- err
			continue
		}

		diff := newDiff(lastChks, chks)
		if diff.Empty() {
			continue
		}

		w.DiffCh <- *diff
	}

	return nil
}
