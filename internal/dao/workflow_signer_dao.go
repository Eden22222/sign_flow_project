package dao

import (
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
