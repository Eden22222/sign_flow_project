package dao

import (
	"errors"
	"fmt"
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

// SelectPageByUserFilters 按当前用户视图与筛选条件分页查询 workflow（Count 与列表使用同一套条件，按 created_at DESC）。
func (d *workflowDaoImpl) SelectPageByUserFilters(userID uint, view string, status string, keyword string, page int, pageSize int) ([]model.WorkflowModel, int64, error) {
	if view != "initiated" && view != "assigned" {
		return nil, 0, fmt.Errorf("unsupported workflow list view")
	}

	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}

	base := d.userListQuery(db, userID, view, status, keyword)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		log.Error(err)
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	workflows := make([]model.WorkflowModel, 0)
	listQ := d.userListQuery(db, userID, view, status, keyword).
		Order("workflow_models.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&workflows)
	if listQ.Error != nil {
		log.Error(listQ.Error)
		return nil, 0, listQ.Error
	}

	return workflows, total, nil
}

func (d *workflowDaoImpl) userListQuery(db *gorm.DB, userID uint, view string, status string, keyword string) *gorm.DB {
	q := db.Model(&model.WorkflowModel{}).
		Joins("INNER JOIN document_models ON document_models.id = workflow_models.document_id AND document_models.deleted_at IS NULL").
		Where("workflow_models.deleted_at IS NULL")

	switch view {
	case "initiated":
		q = q.Where("workflow_models.initiator_id = ?", userID)
	case "assigned":
		q = q.Where(
			`EXISTS (SELECT 1 FROM workflow_signer_models AS ws WHERE ws.workflow_id = workflow_models.id AND ws.signer_id = ? AND ws.deleted_at IS NULL)`,
			userID,
		)
	}

	if status != "" {
		q = q.Where("workflow_models.status = ?", status)
	}

	if keyword != "" {
		pattern := "%" + keyword + "%"
		q = q.Where("(document_models.title ILIKE ? OR document_models.file_name ILIKE ?)", pattern, pattern)
	}

	return q
}
