package dao

import (
	"errors"
	"sign_flow_project/internal/model"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type taskDaoImpl struct{}

var TaskDao = new(taskDaoImpl)

func (d *taskDaoImpl) Create(task *model.TaskModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Create(task)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *taskDaoImpl) CreateTx(tx *gorm.DB, task *model.TaskModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Create(task)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *taskDaoImpl) Update(task *model.TaskModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Save(task)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *taskDaoImpl) UpdateTx(tx *gorm.DB, task *model.TaskModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Save(task)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *taskDaoImpl) SelectByID(id uint) (*model.TaskModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	task := model.TaskModel{}
	res := db.First(&task, id)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &task, nil
}

func (d *taskDaoImpl) SelectCurrentPendingByWorkflowIDTx(tx *gorm.DB, workflowID uint) (*model.TaskModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	task := model.TaskModel{}
	res := tx.
		Where("workflow_id = ? AND status = ?", workflowID, model.TaskStatusPending).
		Order("step_index ASC").
		First(&task)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &task, nil
}

func (d *taskDaoImpl) SelectCurrentPendingByWorkflowID(workflowID uint) (*model.TaskModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	task := model.TaskModel{}
	res := db.
		Where("workflow_id = ? AND status = ?", workflowID, model.TaskStatusPending).
		Order("step_index ASC").
		First(&task)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &task, nil
}

func (d *taskDaoImpl) SelectByWorkflowID(workflowID uint) ([]model.TaskModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var tasks []model.TaskModel
	res := db.
		Where("workflow_id = ?", workflowID).
		Order("step_index ASC").
		Find(&tasks)
	if res.Error != nil {
		log.Error(res.Error)
		return nil, res.Error
	}
	return tasks, nil
}
