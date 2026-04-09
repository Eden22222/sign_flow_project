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

// CountSignersByWorkflowIDs 按 workflow_id 分组统计签署人数量（用于列表批量组装）。
func (d *workflowSignerDaoImpl) CountSignersByWorkflowIDs(workflowIDs []uint) (map[uint]int, error) {
	out := make(map[uint]int)
	if len(workflowIDs) == 0 {
		return out, nil
	}
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}

	type row struct {
		WorkflowID uint  `gorm:"column:workflow_id"`
		Cnt        int64 `gorm:"column:cnt"`
	}
	var rows []row
	res := db.Model(&model.WorkflowSignerModel{}).
		Select("workflow_id, COUNT(*) AS cnt").
		Where("workflow_id IN ?", workflowIDs).
		Group("workflow_id").
		Scan(&rows)
	if res.Error != nil {
		log.Error(res.Error)
		return nil, res.Error
	}
	for _, r := range rows {
		out[r.WorkflowID] = int(r.Cnt)
	}
	return out, nil
}

func (d *workflowSignerDaoImpl) SelectByWorkflowIDTx(tx *gorm.DB, workflowID uint) ([]model.WorkflowSignerModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	var signers []model.WorkflowSignerModel
	res := tx.
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
