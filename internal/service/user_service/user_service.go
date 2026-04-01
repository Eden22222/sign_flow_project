package service

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgconn"
	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/model"

	"gorm.io/gorm"
)

type userServiceImpl struct{}

var UserService = new(userServiceImpl)

type CreateUserRequest struct {
	UserCode string `json:"userCode"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Status   string `json:"status"`
}

type CreateUserResult struct {
	ID       uint   `json:"id"`
	UserCode string `json:"userCode"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Status   string `json:"status"`
}

type UserDetailResult struct {
	ID       uint   `json:"id"`
	UserCode string `json:"userCode"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Status   string `json:"status"`
}

func (s *userServiceImpl) CreateUser(req CreateUserRequest) (*CreateUserResult, error) {
	userCode := strings.TrimSpace(req.UserCode)
	if userCode == "" {
		return nil, fmt.Errorf("userCode is required")
	}
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
		UserCode: userCode,
		Name:     name,
		Email:    strings.TrimSpace(req.Email),
		Avatar:   avatar,
		Status:   status,
	}
	if err := dao.UserDao.Create(u); err != nil {
		if dup, ok := translateUserCodeDuplicateError(err); ok {
			return nil, dup
		}
		return nil, err
	}
	return &CreateUserResult{
		ID:       u.ID,
		UserCode: u.UserCode,
		Name:     u.Name,
		Email:    u.Email,
		Avatar:   u.Avatar,
		Status:   u.Status,
	}, nil
}

// translateUserCodeDuplicateError 将唯一约束冲突转为明确业务错误（避免并发下先查后插误判，且不暴露 500）。
func translateUserCodeDuplicateError(err error) (error, bool) {
	if err == nil {
		return nil, false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return fmt.Errorf("userCode already exists"), true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("userCode already exists"), true
	}
	low := strings.ToLower(err.Error())
	if strings.Contains(low, "duplicate key") ||
		strings.Contains(low, "unique constraint") ||
		strings.Contains(err.Error(), "23505") {
		return fmt.Errorf("userCode already exists"), true
	}
	return nil, false
}

func (s *userServiceImpl) GetByUserCode(userCode string) (*UserDetailResult, error) {
	userCode = strings.TrimSpace(userCode)
	if userCode == "" {
		return nil, fmt.Errorf("userCode is required")
	}
	u, err := dao.UserDao.SelectByUserCode(userCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &UserDetailResult{
		ID:       u.ID,
		UserCode: u.UserCode,
		Name:     u.Name,
		Email:    u.Email,
		Avatar:   u.Avatar,
		Status:   u.Status,
	}, nil
}

func (s *userServiceImpl) BatchGetMapByUserCodes(userCodes []string) (map[string]model.UserModel, error) {
	if len(userCodes) == 0 {
		return map[string]model.UserModel{}, nil
	}
	uniq := make([]string, 0, len(userCodes))
	seen := make(map[string]struct{}, len(userCodes))
	for _, c := range userCodes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		uniq = append(uniq, c)
	}
	if len(uniq) == 0 {
		return map[string]model.UserModel{}, nil
	}
	users, err := dao.UserDao.SelectByUserCodes(uniq)
	if err != nil {
		return nil, err
	}
	m := make(map[string]model.UserModel, len(users))
	for i := range users {
		m[users[i].UserCode] = users[i]
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
