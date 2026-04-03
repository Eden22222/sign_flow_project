package dao

import (
	"errors"
	"sign_flow_project/internal/model"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type userDaoImpl struct{}

var UserDao = new(userDaoImpl)

func (d *userDaoImpl) Create(user *model.UserModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Create(user)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *userDaoImpl) CreateTx(tx *gorm.DB, user *model.UserModel) error {
	if tx == nil {
		return errNilDB
	}
	res := tx.Create(user)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *userDaoImpl) Update(user *model.UserModel) error {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return err
	}
	res := db.Save(user)
	if res.Error != nil {
		log.Error(res.Error)
		return res.Error
	}
	return nil
}

func (d *userDaoImpl) SelectByID(id uint) (*model.UserModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	u := model.UserModel{}
	res := db.First(&u, id)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &u, nil
}

func (d *userDaoImpl) SelectByUserCode(userCode string) (*model.UserModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	u := model.UserModel{}
	res := db.Where("user_code = ?", userCode).First(&u)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &u, nil
}

func (d *userDaoImpl) SelectByUserCodeTx(tx *gorm.DB, userCode string) (*model.UserModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	u := model.UserModel{}
	res := tx.Where("user_code = ?", userCode).First(&u)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &u, nil
}

func (d *userDaoImpl) SelectByEmail(email string) (*model.UserModel, error) {
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	u := model.UserModel{}
	res := db.Where("email = ?", email).First(&u)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &u, nil
}

func (d *userDaoImpl) SelectByEmailTx(tx *gorm.DB, email string) (*model.UserModel, error) {
	if tx == nil {
		return nil, errNilDB
	}
	u := model.UserModel{}
	res := tx.Where("email = ?", email).First(&u)
	if res.Error != nil {
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Error(res.Error)
		}
		return nil, res.Error
	}
	return &u, nil
}

func (d *userDaoImpl) SelectByUserCodes(userCodes []string) ([]model.UserModel, error) {
	if len(userCodes) == 0 {
		return []model.UserModel{}, nil
	}
	db, err := defaultDB()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var users []model.UserModel
	res := db.Where("user_code IN ?", userCodes).Find(&users)
	if res.Error != nil {
		log.Error(res.Error)
		return nil, res.Error
	}
	return users, nil
}
