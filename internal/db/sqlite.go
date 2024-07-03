package db

import (
	"errors"
	"fmt"

	"github.com/blacktop/ipsw/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Sqlite is a database that stores data in a sqlite database.
type Sqlite struct {
	URL string
	// Config
	BatchSize int

	db *gorm.DB
}

// NewSqlite creates a new Sqlite database.
func NewSqlite(path string, batchSize int) (Database, error) {
	if path == "" {
		return nil, fmt.Errorf("'path' is required")
	}
	return &Sqlite{
		URL:       path,
		BatchSize: batchSize,
	}, nil
}

// Connect connects to the database.
func (s *Sqlite) Connect() (err error) {
	s.db, err = gorm.Open(sqlite.Open(s.URL), &gorm.Config{
		CreateBatchSize:        s.BatchSize,
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Error),
		// Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect sqlite database: %w", err)
	}
	return s.db.AutoMigrate(
		&model.Ipsw{},
		&model.Device{},
		&model.Kernelcache{},
		&model.DyldSharedCache{},
		&model.Macho{},
		&model.Symbol{},
	)
}

// Create creates a new entry in the database.
// It returns ErrAlreadyExists if the key already exists.
func (s *Sqlite) Create(value any) error {
	// if result := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(value); result.Error != nil {
	if result := s.db.FirstOrCreate(value); result.Error != nil {
		return result.Error
	}
	return nil
}

// Get returns the value for the given key.
// It returns ErrNotFound if the key does not exist.
func (s *Sqlite) Get(key string) (*model.Ipsw, error) {
	i := &model.Ipsw{}
	s.db.First(&i, key)
	return i, nil
}

// GetByName returns the IPSW for the given name.
// It returns ErrNotFound if the key does not exist.
func (s *Sqlite) GetByName(name string) (*model.Ipsw, error) {
	i := &model.Ipsw{Name: name}
	if result := s.db.First(&i); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, model.ErrNotFound
		}
		return nil, result.Error
	}
	return i, nil
}

func (s *Sqlite) GetSymbol(uuid string, address uint64) (*model.Symbol, error) {
	var symbol model.Symbol
	if err := s.db.Joins("JOIN macho_syms ON macho_syms.symbol_id = symbols.id").
		Joins("JOIN machos ON machos.uuid = macho_syms.macho_uuid").
		Where("machos.uuid = ? AND symbols.start <= ? AND ? < symbols.end", uuid, address, address).
		First(&symbol).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &symbol, nil
}

func (s *Sqlite) GetSymbols(uuid string) ([]*model.Symbol, error) {
	var syms []*model.Symbol
	if err := s.db.Joins("JOIN macho_syms ON macho_syms.symbol_id = symbols.id").
		Joins("JOIN machos ON machos.uuid = macho_syms.macho_uuid").
		Where("machos.uuid = ?", uuid).
		Find(syms).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return syms, nil
}

// Set sets the value for the given key.
// It overwrites any previous value for that key.
func (s *Sqlite) Save(value any) error {
	if result := s.db.Save(value); result.Error != nil {
		return result.Error
	}
	return nil
}

// Delete removes the given key.
// It returns ErrNotFound if the key does not exist.
func (s *Sqlite) Delete(key string) error {
	s.db.Delete(&model.Ipsw{}, key)
	return nil
}

// Close closes the database.
// It returns ErrClosed if the database is already closed.
func (s *Sqlite) Close() error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
