package projects

import (
	"context"
	"database/sql"
	"sync"

	"github.com/google/uuid"
)

type Resolver struct {
	repo *Repo

	mu    sync.RWMutex
	cache map[string]uuid.UUID
}

func NewResolver(db *sql.DB) *Resolver {
	return &Resolver{
		repo:  NewRepo(db),
		cache: make(map[string]uuid.UUID),
	}
}

func (r *Resolver) Resolve(ctx context.Context, repoKey string) (uuid.UUID, error) {
	r.mu.RLock()
	if id, ok := r.cache[repoKey]; ok {
		r.mu.RUnlock()
		return id, nil
	}
	r.mu.RUnlock()

	id, err := r.repo.EnsureProject(ctx, repoKey)
	if err != nil {
		return uuid.UUID{}, err
	}

	r.mu.Lock()
	r.cache[repoKey] = id
	r.mu.Unlock()

	return id, nil
}

