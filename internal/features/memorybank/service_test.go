package memorybank

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

type fakeProjects struct {
	id uuid.UUID
}

func (f fakeProjects) Resolve(ctx context.Context, repoKey string) (uuid.UUID, error) {
	return f.id, nil
}

func TestService_RulesList_GlobalOnlyWhenNoRepoKey(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	svc := NewService(repo, fakeProjects{id: uuid.New()})

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id, project_id, scope, priority, title, content, enabled, updated_at
FROM mb_rules
WHERE project_id IS NULL AND enabled = true
ORDER BY priority ASC, project_id NULLS LAST, id ASC
LIMIT $1 OFFSET $2`)).
		WithArgs(200, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "scope", "priority", "title", "content", "enabled", "updated_at"}))

	_, err = svc.ListRules(context.Background(), nil, "", true, 200, 0)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

