package dao

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"sign_flow_project/internal/model"

	"gorm.io/gorm"
)

type workflowSignerDaoImpl struct{}

var WorkflowSignerDao = new(workflowSignerDaoImpl)

func (d *workflowSignerDaoImpl) Create(signers []*model.WorkflowSignerModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Create(&signers)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *workflowSignerDaoImpl) CreateTx(tx *gorm.DB, signers []*model.WorkflowSignerModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Create(&signers)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *workflowSignerDaoImpl) SelectByWorkflowIDAndStepIndexTx(tx *gorm.DB, workflowID uint, stepIndex int) (*model.WorkflowSignerModel, error) {
	var signer model.WorkflowSignerModel
	err := tx.Where("workflow_id = ? AND step_index = ?", workflowID, stepIndex).First(&signer).Error
	if err != nil {
		return nil, err
	}
	return &signer, nil
}

func (d *workflowSignerDaoImpl) SelectByWorkflowID(workflowID uint) ([]model.WorkflowSignerModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var signers []model.WorkflowSignerModel
	res := db.
		Where("workflow_id = ?", workflowID).
		Order("step_index ASC").
		Find(&signers)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return signers, nil
}
