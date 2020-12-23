package gormgoose

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kylelemons/go-gypsy/yaml"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DBDriver encapsulates the info needed to work with
// a specific database driver
type DBDriver struct {
	Name    string
	OpenStr string
	Import  string
}

type DBConf struct {
	MigrationsDir string
	Env           string
	Driver        DBDriver
	PgSchema      string
}

// NewDBConf extract configuration details from the given file
func NewDBConf(p, env string, pgschema string) (*DBConf, error) {

	cfgFile := filepath.Join(p, "dbconf.yml")

	f, err := yaml.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	drv, err := f.Get(fmt.Sprintf("%s.driver", env))
	if err != nil {
		return nil, err
	}
	drv = os.ExpandEnv(drv)

	open, err := f.Get(fmt.Sprintf("%s.open", env))
	if err != nil {
		return nil, err
	}
	open = os.ExpandEnv(open)

	d := newDBDriver(drv, open)

	// allow the configuration to override the Import for this driver
	if imprt, err := f.Get(fmt.Sprintf("%s.import", env)); err == nil {
		d.Import = imprt
	}

	if !d.IsValid() {
		return nil, fmt.Errorf("Invalid DBConf: %v", d)
	}

	return &DBConf{
		MigrationsDir: filepath.Join(p, "migrations"),
		Env:           env,
		Driver:        d,
		PgSchema:      pgschema,
	}, nil
}

// Create a new DBDriver and populate driver specific
// fields for drivers that we know about.
// Further customization may be done in NewDBConf
func newDBDriver(name, open string) DBDriver {

	d := DBDriver{
		Name:    name,
		OpenStr: open,
	}

	switch name {
	case "postgres":
		d.Import = "gorm.io/driver/postgres"

	case "mysql":
		d.Import = "gorm.io/driver/mysql"
		d.OpenStr = d.OpenStr + "?charset=utf8&parseTime=True&loc=Local"

	case "sqlite3":
		d.Import = "gorm.io/driver/sqlite"
	}

	return d
}

// IsValid ensure we have enough info about this driver
func (drv *DBDriver) IsValid() bool {
	return len(drv.Import) > 0
}

// OpenDBFromDBConf wraps database/sql.DB.Open() and configures
// the newly opened DB based on the given DBConf.
//
// Callers must Close() the returned DB.
func OpenDBFromDBConf(conf *DBConf) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(conf.Driver.OpenStr), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// if a postgres schema has been specified, apply it
	if conf.Driver.Name == "postgres" && conf.PgSchema != "" {
		if err := db.Exec("SET search_path TO " + conf.PgSchema).Error; err != nil {
			return nil, err
		}
	}

	return db, nil
}
