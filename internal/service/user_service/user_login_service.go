package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	ID       uint   `json:"id"`
	UserCode string `json:"userCode"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Status   string `json:"status"`
}

type AuthResult struct {
	User        *AuthUserResult `json:"user"`
	AccessToken string          `json:"accessToken"`
}

type jwtAccessClaims struct {
	UserID   uint   `json:"user_id"`
	UserCode string `json:"user_code"`
	Email    string `json:"email"`
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

	userCode, err := allocateUserCode(name)
	if err != nil {
		return nil, err
	}

	u := &model.UserModel{
		UserCode:     userCode,
		Name:         name,
		Email:        email,
		Avatar:       DefaultAvatarFromName(name),
		Status:       "active",
		PasswordHash: string(hash),
	}
	if err := dao.UserDao.Create(u); err != nil {
		if dup, ok := translateUserCodeDuplicateError(err); ok {
			return nil, dup
		}
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
		UserID:   u.ID,
		UserCode: u.UserCode,
		Email:    u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(jwtSecret)
}

// ParseAccessToken 校验 JWT 并返回 claims 中的用户标识（供 auth 中间件使用）。
func (s *userLoginServiceImpl) ParseAccessToken(tokenStr string) (userID uint, userCode, email string, err error) {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return 0, "", "", ErrInvalidToken
	}
	var claims jwtAccessClaims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, "", "", ErrInvalidToken
	}
	return claims.UserID, claims.UserCode, claims.Email, nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func toAuthUserResult(u *model.UserModel) *AuthUserResult {
	if u == nil {
		return nil
	}
	return &AuthUserResult{
		ID:       u.ID,
		UserCode: u.UserCode,
		Name:     u.Name,
		Email:    u.Email,
		Avatar:   u.Avatar,
		Status:   u.Status,
	}
}

func allocateUserCode(name string) (string, error) {
	base := userCodeSlugFromName(name)
	for i := 0; i < 12; i++ {
		code := base
		if i > 0 {
			code = base + "_" + randomHexSuffix()
		}
		_, err := dao.UserDao.SelectByUserCode(code)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return code, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("could not allocate userCode")
}

func userCodeSlugFromName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	var b strings.Builder
	prevUnderscore := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUnderscore = false
		case r == ' ', r == '_', r == '-':
			if b.Len() > 0 && !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" {
		s = "user"
	}
	return "u_" + s
}

func randomHexSuffix() string {
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}

// translateUserCodeDuplicateError 识别 user_code 唯一键冲突（GORM / PostgreSQL 23505 等），返回友好业务错误。
// 与 CreateUser 共用，避免注册/管理接口暴露原始 DB 错误。
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
