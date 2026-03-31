package dao

import (
	"errors"
	"sign_flow_project/internal/model"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type workflowDaoImpl struct{}

var WorkflowDao = new(workflowDaoImpl)

func (d *workflowDaoImpl) Create(workflow *model.WorkflowModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Create(workflow)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *workflowDaoImpl) CreateTx(tx *gorm.DB, workflow *model.WorkflowModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Create(workflow)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *workflowDaoImpl) Update(workflow *model.WorkflowModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Save(workflow)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *workflowDaoImpl) UpdateTx(tx *gorm.DB, workflow *model.WorkflowModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Save(workflow)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *workflowDaoImpl) SelectByID(id uint) (*model.WorkflowModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	workflow := model.WorkflowModel{}
	res := db.First(&workflow, id)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &workflow, nil
}

func (d *workflowDaoImpl) SelectByIDTx(tx *gorm.DB, id uint) (*model.WorkflowModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	workflow := model.WorkflowModel{}
	res := tx.First(&workflow, id)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &workflow, nil
}

func (d *workflowDaoImpl) SelectPage(page int, pageSize int) ([]model.WorkflowModel, int64, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}

	var total int64
	countRes := db.Model(&model.WorkflowModel{}).Count(&total)
	if countRes.Error != nil {
		log.Error(countRes.Error)
		return nil, 0, countRes.Error
	}

	offset := (page - 1) * pageSize
	workflows := make([]model.WorkflowModel, 0)
	listRes := db.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&workflows)
	if listRes.Error != nil {
		log.Error(listRes.Error)
		return nil, 0, listRes.Error
	}

	return workflows, total, nil
}
