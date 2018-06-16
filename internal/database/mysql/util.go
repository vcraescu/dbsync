package mysql

import (
	"fmt"
	"strconv"
	"os/exec"
	"bytes"
	"errors"
	"strings"
	"path/filepath"
)

func generateDSN(username, password, host string, port int, dbname string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", username, password, host, port, dbname)
}

func generateDropTableStatement(table string) string {
	return fmt.Sprintf("drop table if exists `%s`", table)
}

func mysqlDump(username, password, host string, port int, schema string, tables ...string) (string, error) {
	args := []string{
		"-h",
		host,
		"-P",
		strconv.Itoa(port),
		"-u",
		username,
		fmt.Sprintf("-p%s", password),
		schema,
	}
	args = append(args, tables...)

	path, err := exec.LookPath("mysqldump")
	if err != nil {
		path, err = filepath.Abs("bin/mysqldump")
		if err != nil {
			return "", errors.New("mysqldump not found")
		}
	}

	cmd := exec.Command(path, args...)

	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", errors.New(fmt.Sprintf("%s: %s", err, stderr.String()))
	}

	return out.String(), nil
}

func mysqlImport(username, password, host string, port int, schema, dump string) (string, error) {
	args := []string{
		"-h",
		host,
		"-P",
		strconv.Itoa(port),
		"-u",
		username,
		fmt.Sprintf("-p%s", password),
		schema,
	}

	path, err := exec.LookPath("mysql")
	if err != nil {
		path, err = filepath.Abs("bin/mysql")
		if err != nil {
			return "", errors.New("mysql client not found")
		}
	}

	cmd := exec.Command(path, args...)

	var out bytes.Buffer
	var stderr bytes.Buffer
	var stdin bytes.Buffer

	stdin.WriteString(dump)

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	cmd.Stdin = &stdin
	if err := cmd.Run(); err != nil {
		return "", errors.New(fmt.Sprintf("%s: %s", err, stderr.String()))
	}

	return out.String(), nil
}

func compressMySQLDump(sql string) string {
	var newSQL []string
	for _, l := range strings.Split(sql, "\n") {
		l = strings.Trim(l, " ")
		if strings.HasPrefix(l, "--") || l == "" {
			continue
		}

		newSQL = append(newSQL, l)
	}

	return strings.Join(newSQL, "\n")
}
