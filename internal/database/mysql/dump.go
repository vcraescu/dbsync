package mysql

type Dumper struct {
	cfg ConnectionConfig
}

func NewDumper(cfg ConnectionConfig) *Dumper {
	return &Dumper{
		cfg: cfg,
	}
}

func (d *Dumper) DumpTables(tables ...string) (string, error) {
	out, err := mysqlDump(
		d.cfg.Username,
		d.cfg.Password,
		d.cfg.Host,
		d.cfg.Port,
		d.cfg.Schema,
		tables...
	)
	if err != nil {
		return "", err
	}

	return compressMySQLDump(out), nil
}
