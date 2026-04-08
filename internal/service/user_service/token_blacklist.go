package service

import (
	"sync"
	"time"
)

// TokenBlacklistStore 记录已注销的 access token，直至其 JWT exp。
// 默认使用内存实现；后续可替换为 Redis（TTL=剩余有效期），接口保持不变。
type TokenBlacklistStore interface {
	Revoke(token string, expiresAt time.Time)
	IsRevoked(token string) bool
}

// TokenBlacklist 全局黑名单存储，可在集成测试或启动时替换为 Redis 实现。
var TokenBlacklist TokenBlacklistStore = NewInMemoryTokenBlacklistStore()

type inMemoryTokenBlacklistStore struct {
	mu      sync.RWMutex
	revoked map[string]time.Time
}

func NewInMemoryTokenBlacklistStore() TokenBlacklistStore {
	return &inMemoryTokenBlacklistStore{
		revoked: make(map[string]time.Time),
	}
}

func (s *inMemoryTokenBlacklistStore) Revoke(token string, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[token] = expiresAt
	s.purgeExpiredLocked()
}

func (s *inMemoryTokenBlacklistStore) IsRevoked(token string) bool {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.revoked[token]
	if !ok {
		s.purgeExpiredLocked()
		return false
	}
	if !now.Before(exp) {
		delete(s.revoked, token)
		s.purgeExpiredLocked()
		return false
	}
	return true
}

func (s *inMemoryTokenBlacklistStore) purgeExpiredLocked() {
	now := time.Now()
	for t, exp := range s.revoked {
		if !now.Before(exp) {
			delete(s.revoked, t)
		}
	}
}
