package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// jwtSecret 为开发阶段临时密钥，用于 HS256 签发 access token。
// 生产环境应改为可配置密钥并妥善保管；当前项目暂不引入完整配置系统。
var jwtSecret = []byte("sign-flow-project-secret")

const accessTokenTTL = 24 * time.Hour

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type userLoginServiceImpl struct{}

var UserLoginService = new(userLoginServiceImpl)

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthUserResult struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
	Status string `json:"status"`
}

type AuthResult struct {
	User        *AuthUserResult `json:"user"`
	AccessToken string          `json:"accessToken"`
}

type jwtAccessClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *userLoginServiceImpl) Register(req RegisterRequest) (*AuthResult, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	password := strings.TrimSpace(req.Password)
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	if _, err := dao.UserDao.SelectByEmail(email); err == nil {
		return nil, fmt.Errorf("email already registered")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &model.UserModel{
		Name:         name,
		Email:        email,
		Avatar:       DefaultAvatarFromName(name),
		Status:       "active",
		PasswordHash: string(hash),
	}
	if err := dao.UserDao.Create(u); err != nil {
		return nil, err
	}

	token, err := s.signAccessToken(u)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		User:        toAuthUserResult(u),
		AccessToken: token,
	}, nil
}

func (s *userLoginServiceImpl) Login(req LoginRequest) (*AuthResult, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, ErrInvalidCredentials
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return nil, ErrInvalidCredentials
	}

	u, err := dao.UserDao.SelectByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if u.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.signAccessToken(u)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		User:        toAuthUserResult(u),
		AccessToken: token,
	}, nil
}

func (s *userLoginServiceImpl) GetMe(userID uint) (*AuthUserResult, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user not found")
	}
	u, err := dao.UserDao.SelectByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return toAuthUserResult(u), nil
}

func (s *userLoginServiceImpl) signAccessToken(u *model.UserModel) (string, error) {
	if u == nil {
		return "", fmt.Errorf("user is nil")
	}
	now := time.Now()
	claims := jwtAccessClaims{
		UserID: u.ID,
		Email:  u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(jwtSecret)
}

// ParseAccessToken 校验 JWT 并返回 claims 中的用户标识（供 auth 中间件使用）。
func (s *userLoginServiceImpl) ParseAccessToken(tokenStr string) (userID uint, email string, err error) {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return 0, "", ErrInvalidToken
	}
	var claims jwtAccessClaims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, "", ErrInvalidToken
	}
	return claims.UserID, claims.Email, nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func toAuthUserResult(u *model.UserModel) *AuthUserResult {
	if u == nil {
		return nil
	}
	return &AuthUserResult{
		ID:     u.ID,
		Name:   u.Name,
		Email:  u.Email,
		Avatar: u.Avatar,
		Status: u.Status,
	}
}
