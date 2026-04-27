package service

import (
	"context"
	"errors"

	"whisper/backend/internal/repository"
)

func (a *App) SeedDemo(ctx context.Context) error {
	if _, err := a.store.FindUserByEmail(ctx, "admin@demo.local"); err == nil {
		return nil
	} else if !repository.IsNotFound(err) {
		return err
	}

	_, _, err := a.RegisterCompany(ctx, "Whisper Demo", "", "Admin Demo", "admin@demo.local", "admin12345")
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
