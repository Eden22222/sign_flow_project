package dao

import (
	"errors"
	"sign_flow_project/internal/model"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type documentVersionDaoImpl struct{}

var DocumentVersionDao = new(documentVersionDaoImpl)

func (d *documentVersionDaoImpl) Create(v *model.DocumentVersionModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Create(v)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *documentVersionDaoImpl) CreateTx(tx *gorm.DB, v *model.DocumentVersionModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Create(v)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *documentVersionDaoImpl) SelectByID(id uint) (*model.DocumentVersionModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var row model.DocumentVersionModel
	res := db.First(&row, id)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &row, nil
}

func (d *documentVersionDaoImpl) SelectByIDTx(tx *gorm.DB, id uint) (*model.DocumentVersionModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	var row model.DocumentVersionModel
	res := tx.First(&row, id)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &row, nil
}

func (d *documentVersionDaoImpl) SelectByDocumentID(documentID uint) ([]model.DocumentVersionModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var rows []model.DocumentVersionModel
	res := db.Where("document_id = ?", documentID).
		Order("version_no DESC").
		Find(&rows)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return rows, nil
}

func (d *documentVersionDaoImpl) SelectByDocumentIDTx(tx *gorm.DB, documentID uint) ([]model.DocumentVersionModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	var rows []model.DocumentVersionModel
	res := tx.Where("document_id = ?", documentID).
		Order("version_no DESC").
		Find(&rows)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return rows, nil
}

func (d *documentVersionDaoImpl) SelectLatestByDocumentID(documentID uint) (*model.DocumentVersionModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var row model.DocumentVersionModel
	res := db.Where("document_id = ?", documentID).
		Order("version_no DESC").
		First(&row)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &row, nil
}

func (d *documentVersionDaoImpl) SelectLatestByDocumentIDTx(tx *gorm.DB, documentID uint) (*model.DocumentVersionModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	var row model.DocumentVersionModel
	res := tx.Where("document_id = ?", documentID).
		Order("version_no DESC").
		First(&row)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &row, nil
}
