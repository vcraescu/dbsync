package mysql

// Importer - importer class
type Importer struct {
	cfg ConnectionConfig
}

// NewImporter - import constructor
func NewImporter(cfg ConnectionConfig) *Importer {
	return &Importer{
		cfg: cfg,
	}
}

// Import - import sql dump
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
