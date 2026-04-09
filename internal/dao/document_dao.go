package dao

import (
	"errors"
	"sign_flow_project/internal/model"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type documentDaoImpl struct{}

var DocumentDao = new(documentDaoImpl)

// 普通创建（使用 InitPostgres 后的全局连接）
func (d *documentDaoImpl) Create(document *model.DocumentModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Create(document)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

// 事务创建（传入 tx）
func (d *documentDaoImpl) CreateTx(tx *gorm.DB, document *model.DocumentModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Create(document)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *documentDaoImpl) Update(document *model.DocumentModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Save(document)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *documentDaoImpl) UpdateTx(tx *gorm.DB, document *model.DocumentModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Save(document)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *documentDaoImpl) SelectByID(id uint) (*model.DocumentModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	document := model.DocumentModel{}
	res := db.First(&document, id)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &document, nil
}

// SelectByIDs 批量按主键查询文档（列表等场景批量加载，顺序不保证）。
func (d *documentDaoImpl) SelectByIDs(ids []uint) ([]model.DocumentModel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var documents []model.DocumentModel
	res := db.Where("id IN ?", ids).Find(&documents)
	if res.Error != nil {
		log.Error(res.Error)
		return nil, res.Error
	}
	return documents, nil
}

func (d *documentDaoImpl) SelectByIDTx(tx *gorm.DB, id uint) (*model.DocumentModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	document := model.DocumentModel{}
	res := tx.First(&document, id)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &document, nil
}
