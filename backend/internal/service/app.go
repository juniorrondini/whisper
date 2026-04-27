package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"whisper/backend/internal/auth"
	"whisper/backend/internal/config"
	"whisper/backend/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg   config.Config
	store *repository.Store
	redis *redis.Client
}

type TokenPair struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	User                  repository.User `json:"user"`
}

func NewApp(cfg config.Config, store *repository.Store, redis *redis.Client) *App {
	return &App{cfg: cfg, store: store, redis: redis}
}

func (a *App) Store() *repository.Store {
	return a.store
}

func (a *App) Redis() *redis.Client {
	return a.redis
}

func (a *App) RegisterCompany(ctx context.Context, companyName, cnpj, adminName, email, password string) (repository.Company, TokenPair, error) {
	if strings.TrimSpace(companyName) == "" || strings.TrimSpace(adminName) == "" || strings.TrimSpace(email) == "" || len(password) < 8 {
		return repository.Company{}, TokenPair{}, errors.New("company, admin, email and password with at least 8 chars are required")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return repository.Company{}, TokenPair{}, err
	}
	company, user, err := a.store.CreateCompanyWithAdmin(ctx, companyName, cnpj, adminName, email, hash)
	if err != nil {
		return repository.Company{}, TokenPair{}, err
	}
	tokens, err := a.issueTokens(ctx, user)
	return company, tokens, err
}

func (a *App) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := a.store.FindUserByEmail(ctx, email)
	if err != nil || !auth.CheckPassword(user.PasswordHash, password) || user.Status != "active" {
		return TokenPair{}, errors.New("invalid credentials")
	}
	return a.issueTokens(ctx, user.User)
}

func (a *App) Refresh(ctx context.Context, rawRefreshToken string) (TokenPair, error) {
	if rawRefreshToken == "" {
		return TokenPair{}, errors.New("refresh token is required")
	}
	user, err := a.store.FindRefreshTokenUser(ctx, auth.HashRefreshToken(rawRefreshToken))
	if err != nil || user.Status != "active" {
		return TokenPair{}, errors.New("invalid refresh token")
	}
	_ = a.store.RevokeRefreshToken(ctx, auth.HashRefreshToken(rawRefreshToken))
	return a.issueTokens(ctx, user.User)
}

func (a *App) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}
	return a.store.RevokeRefreshToken(ctx, auth.HashRefreshToken(rawRefreshToken))
}

func (a *App) CreateUser(ctx context.Context, companyID uuid.UUID, name, email, role, password string) (repository.User, error) {
	if name == "" || email == "" || len(password) < 8 {
		return repository.User{}, errors.New("name, email and password with at least 8 chars are required")
	}
	if !validRole(role) {
		return repository.User{}, errors.New("invalid role")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return repository.User{}, err
	}
	return a.store.CreateUser(ctx, companyID, name, email, role, hash)
}

func (a *App) CreateMessage(ctx context.Context, companyID uuid.UUID, conversationID string, senderType string, senderID *uuid.UUID, content string) (repository.Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return repository.Message{}, errors.New("message content is required")
	}
	if len(content) > 4000 {
		return repository.Message{}, errors.New("message content is too long")
	}
	return a.store.CreateMessage(ctx, companyID, conversationID, senderType, senderID, content)
}

func (a *App) issueTokens(ctx context.Context, user repository.User) (TokenPair, error) {
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return TokenPair{}, err
	}
	companyID, err := uuid.Parse(user.CompanyID)
	if err != nil {
		return TokenPair{}, err
	}
	access, accessExp, err := auth.GenerateAccessToken(a.cfg.JWTSecret, a.cfg.AccessTokenDuration, userID, companyID, user.Role)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, refreshHash, err := auth.GenerateRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	refreshExp := time.Now().UTC().Add(a.cfg.RefreshTokenDuration)
	if err := a.store.CreateRefreshToken(ctx, userID, refreshHash, refreshExp); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:           access,
		AccessTokenExpiresAt:  accessExp,
		RefreshToken:          refresh,
		RefreshTokenExpiresAt: refreshExp,
		User:                  user,
	}, nil
}

func validRole(role string) bool {
	switch role {
	case "ADMIN", "SUPERVISOR", "ATENDENTE", "VISUALIZADOR":
		return true
	default:
		return false
	}
}
