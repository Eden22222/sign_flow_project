package model

type UserModel struct {
	Model
	UserCode string `json:"userCode" gorm:"column:user_code;type:varchar(50);uniqueIndex;not null"`
	Name     string `json:"name" gorm:"column:name;type:varchar(100);not null"`
	Email    string `json:"email" gorm:"column:email;type:varchar(255)"`
	Avatar   string `json:"avatar" gorm:"column:avatar;type:varchar(20)"`
	Status   string `json:"status" gorm:"column:status;type:varchar(50);not null;default:'active'"`
}

func (UserModel) TableName() string {
	return "users"
}
