package mysql

type Importer struct {
	cfg ConnectionConfig
}

func NewImporter(cfg ConnectionConfig) *Importer {
	return &Importer{
		cfg: cfg,
	}
}

func (imp *Importer) Import(dump string) error {
	_, err := mysqlImport(
		imp.cfg.Username,
		imp.cfg.Password,
		imp.cfg.Host,
		imp.cfg.Port,
		imp.cfg.Schema,
		dump,
	)

	return err
}
