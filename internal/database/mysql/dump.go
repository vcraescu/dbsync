package mysql

// Dumper - dumps database sql
type Dumper struct {
	cfg ConnectionConfig
}

// NewDumper - constructor
func NewDumper(cfg ConnectionConfig) *Dumper {
	return &Dumper{
		cfg: cfg,
	}
}

// DumpTables - dump tables sql
func (d *Dumper) DumpTables(tables ...string) (string, error) {
	out, err := mysqlDump(
		d.cfg.Username,
		d.cfg.Password,
		d.cfg.Host,
		d.cfg.Port,
		d.cfg.Schema,
		tables...,
	)
	if err != nil {
		return "", err
	}

	return compressMySQLDump(out), nil
}
