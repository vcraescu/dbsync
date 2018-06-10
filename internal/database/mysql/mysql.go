package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"errors"
	"strings"
)

const driverName = "mysql"

type ConnectionConfig struct {
	Username string
	Password string
	Port     int
	Host     string
	Schema   string
}

// Connection - mysql connection
type Connection struct {
	db  *sql.DB
	cfg ConnectionConfig
}

// New - creates new mysql connection.
func New(cfg ConnectionConfig) *Connection {
	conn := &Connection{
		cfg: cfg,
	}

	return conn
}

func (conn *Connection) isOpened() bool {
	return conn.db != nil
}

// Open - open connection
func (conn *Connection) Open() error {
	if conn.isOpened() {
		return nil
	}

	db, err := sql.Open(
		driverName,
		generateDSN(
			conn.cfg.Username,
			conn.cfg.Password,
			conn.cfg.Host,
			conn.cfg.Port,
			conn.cfg.Schema,
		),
	)

	conn.db = db

	return err
}

// Close - close mysql connection
func (conn *Connection) Close() error {
	if conn.db == nil {
		return nil
	}

	return conn.db.Close()
}

// TableNames - returns table names
func (conn *Connection) TableNames() ([]string, error) {
	rows, err := conn.db.Query("show tables")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		names = append(names, name)
	}

	return names, nil
}

// TableChecksum - returns table checksum
func (conn *Connection) TableChecksum(table string) (string, error) {
	rows, err := conn.db.Query(fmt.Sprintf("select * from %s limit 1", table))
	if err != nil {
		return "", errors.New(fmt.Sprintf("Table Checksum (%s): %s", table, err))
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Table Checksum (%s): %s", table, err))
	}

	q := fmt.Sprintf(
		"select ifnull(md5(group_concat(`%s`)), '') as `hash` from `%s`",
		strings.Join(cols, "`, `"),
		table,
	)
	rows, err = conn.db.Query(q)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Table Checksum (%s): %s", table, err))
	}

	var checksum string
	for rows.Next() {
		if err := rows.Scan(&checksum); err != nil {
			return "", errors.New(fmt.Sprintf("Table Checksum (%s): %s", table, err))
		}
	}

	return checksum, nil
}

// TableChecksums - returns checksums of all the tables
func (conn *Connection) TableChecksums() (map[string]string, error) {
	names, err := conn.TableNames()
	if err != nil {
		return nil, err
	}

	chks := make(map[string]string, len(names))
	for _, name := range names {
		checksum, err := conn.TableChecksum(name)
		if err != nil {
			return nil, err
		}

		chks[name] = checksum
	}

	return chks, nil
}
