package service

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/model"

	"gorm.io/gorm"
)

type userServiceImpl struct{}

var UserService = new(userServiceImpl)

type CreateUserRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
	Status string `json:"status"`
}

type CreateUserResult struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
	Status string `json:"status"`
}

type UserDetailResult struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
	Status string `json:"status"`
}

func (s *userServiceImpl) CreateUser(req CreateUserRequest) (*CreateUserResult, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	avatar := strings.TrimSpace(req.Avatar)
	if avatar == "" {
		avatar = defaultAvatarFromName(name)
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}

	u := &model.UserModel{
		Name:     name,
		Email:    strings.TrimSpace(req.Email),
		Avatar:   avatar,
		Status:   status,
	}
	if err := dao.UserDao.Create(u); err != nil {
		return nil, err
	}
	return &CreateUserResult{
		ID:     u.ID,
		Name:   u.Name,
		Email:  u.Email,
		Avatar: u.Avatar,
		Status: u.Status,
	}, nil
}

func (s *userServiceImpl) GetByID(id uint) (*UserDetailResult, error) {
	if id == 0 {
		return nil, fmt.Errorf("id is required")
	}
	u, err := dao.UserDao.SelectByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &UserDetailResult{
		ID:     u.ID,
		Name:   u.Name,
		Email:  u.Email,
		Avatar: u.Avatar,
		Status: u.Status,
	}, nil
}

func (s *userServiceImpl) BatchGetMapByIDs(userIDs []uint) (map[uint]model.UserModel, error) {
	if len(userIDs) == 0 {
		return map[uint]model.UserModel{}, nil
	}
	uniq := make([]uint, 0, len(userIDs))
	seen := make(map[uint]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return map[uint]model.UserModel{}, nil
	}
	users, err := dao.UserDao.SelectByIDs(uniq)
	if err != nil {
		return nil, err
	}
	m := make(map[uint]model.UserModel, len(users))
	for i := range users {
		m[users[i].ID] = users[i]
	}
	return m, nil
}

// defaultAvatarFromName：英文多词取词首字母；单词取前两字母；含汉字时取连续汉字 1～2 字。
func defaultAvatarFromName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	rs := []rune(name)
	var han []rune
	for _, r := range rs {
		if unicode.Is(unicode.Han, r) {
			han = append(han, r)
		}
	}
	if len(han) >= 1 {
		if len(han) == 1 {
			return string(han[0])
		}
		return string(han[:2])
	}

	words := strings.Fields(name)
	if len(words) == 0 {
		return "?"
	}
	if len(words) >= 2 {
		var b strings.Builder
		for _, w := range words {
			for _, ch := range []rune(w) {
				if isLatinLetter(ch) {
					b.WriteRune(unicode.ToUpper(ch))
					break
				}
			}
		}
		if b.Len() > 0 {
			return b.String()
		}
	}

	letters := extractLatinLettersUpper(words[0])
	if len(letters) == 0 {
		return string(rs[0])
	}
	if len(letters) == 1 {
		return string(letters[0])
	}
	return string([]rune{letters[0], letters[1]})
}

func extractLatinLettersUpper(w string) []rune {
	var out []rune
	for _, ch := range []rune(w) {
		if isLatinLetter(ch) {
			out = append(out, unicode.ToUpper(ch))
		}
	}
	return out
}

func isLatinLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// DefaultAvatarFromName 供注册等流程复用，与 CreateUser 默认头像规则一致。
func DefaultAvatarFromName(name string) string {
	return defaultAvatarFromName(name)
}
