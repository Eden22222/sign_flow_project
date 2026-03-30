package dao

import (
	"errors"
	"sign_flow_project/internal/model"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type documentFieldDaoImpl struct{}

var DocumentFieldDao = new(documentFieldDaoImpl)

func (d *documentFieldDaoImpl) Create(field *model.DocumentFieldModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Create(field)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *documentFieldDaoImpl) CreateTx(tx *gorm.DB, field *model.DocumentFieldModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Create(field)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *documentFieldDaoImpl) BatchCreateTx(tx *gorm.DB, fields []*model.DocumentFieldModel) error {
	if tx == nil {
		return errNilDB
	}
	if len(fields) == 0 {
		return nil
	}
	res := tx.Create(&fields)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *documentFieldDaoImpl) DeleteByWorkflowIDTx(tx *gorm.DB, workflowID uint) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Where("workflow_id = ?", workflowID).Delete(&model.DocumentFieldModel{})
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *documentFieldDaoImpl) SelectByWorkflowID(workflowID uint) ([]model.DocumentFieldModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var fields []model.DocumentFieldModel
	res := db.Where("workflow_id = ?", workflowID).Find(&fields)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return fields, nil
}

func (d *documentFieldDaoImpl) SelectByWorkflowIDTx(tx *gorm.DB, workflowID uint) ([]model.DocumentFieldModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	var fields []model.DocumentFieldModel
	res := tx.Where("workflow_id = ?", workflowID).Find(&fields)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return fields, nil
}

