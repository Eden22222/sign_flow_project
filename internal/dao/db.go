package dao

import (
	"errors"

	infradb "sign_flow_project/internal/infra/db"
	"gorm.io/gorm"
)

var (
	errDBNotInitialized = errors.New("database not initialized")
	errNilDB            = errors.New("dao: nil *gorm.DB")
)

func defaultDB() (*gorm.DB, error) {
	g := infradb.GetPostgres()
	if g == nil {
		return nil, errDBNotInitialized
	}
	return g, nil
}