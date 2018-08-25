package mysql

import (
	"errors"
	"fmt"
	"strings"
)

// Diff - computed diff
type Diff struct {
	Create []string
	Delete []string
}

// Empty - returns true if diff is empty
func (d *Diff) Empty() bool {
	return len(d.Create) == 0 && len(d.Delete) == 0
}

// GenerateSQL - generate dump sql
func (d *Diff) GenerateSQL(dumper *Dumper) (string, error) {
	var dump string
	if d.Empty() {
		return dump, errors.New("diff empty")
	}

	if len(d.Create) > 0 {
		var err error
		dump, err = dumper.DumpTables(d.Create...)
		if err != nil {
			return dump, fmt.Errorf("Generate SQL: %s", err)
		}

		dump += "\n"
	}

	for _, table := range d.Delete {
		dump += generateDropTableStatement(table) + ";\n"
	}

	return strings.Trim(dump, " \n"), nil
}

// GenerateDiff - generate diff between to databases
func GenerateDiff(masterConn *Connection, slaveConn *Connection) (*Diff, error) {
	masterChecksums, err := getTableChecksums(masterConn)
	if err != nil {
		return nil, fmt.Errorf("master table checksums: %s", err)
	}

	slaveChecksums, err := getTableChecksums(slaveConn)
	if err != nil {
		return nil, fmt.Errorf("slave table checksums: %s", err)
	}

	diff := &Diff{}

	for mt, mc := range masterChecksums {
		sc, ok := slaveChecksums[mt]
		if ok && sc == mc {
			continue
		}

		diff.Create = append(diff.Create, mt)
	}

	for st := range slaveChecksums {
		mc, ok := masterChecksums[st]
		if ok {
			continue
		}

		diff.Delete = append(diff.Delete, mc)
	}

	return diff, nil
}

func getTableChecksums(conn *Connection) (map[string]string, error) {
	if err := conn.Open(); err != nil {
		return nil, err
	}

	return conn.TableChecksums()
}
