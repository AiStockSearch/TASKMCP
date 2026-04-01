package memorybank

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRepo_UpsertDocument_IdempotentByHash(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	r := NewRepo(db)
	projectID := uuid.New()
	docID := uuid.New()

	docKey := "memory-bank/activeContext.md"
	docType := "activeContext"
	title := "Active"
	content := "hello"
	h := hashContent(content)

	// Begin tx
	mock.ExpectBegin()
	// Select document FOR UPDATE returns existing doc with current_version=1
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id, current_version
FROM mb_documents
WHERE project_id = $1 AND doc_key = $2
FOR UPDATE
`)).
		WithArgs(projectID, docKey).
		WillReturnRows(sqlmock.NewRows([]string{"id", "current_version"}).AddRow(docID, 1))

	// Select current version hash equals new hash
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT content_hash
FROM mb_document_versions
WHERE document_id = $1 AND version = $2
`)).
		WithArgs(docID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"content_hash"}).AddRow(h))

	// Update metadata (best-effort) and commit
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE mb_documents
SET doc_type = $3, title = NULLIF($4, ''), updated_at = now()
WHERE id = $1 AND project_id = $2
`)).
		WithArgs(docID, projectID, docType, title).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	out, err := r.UpsertDocument(context.Background(), projectID, docKey, docType, title, content)
	if err != nil {
		t.Fatalf("UpsertDocument: %v", err)
	}
	if out.DocID != docID.String() {
		t.Fatalf("DocID mismatch: got %s want %s", out.DocID, docID.String())
	}
	if out.Version != 1 {
		t.Fatalf("Version mismatch: got %d want %d", out.Version, 1)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRepo_UpsertDocument_NewVersionWhenHashChanges(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	r := NewRepo(db)
	projectID := uuid.New()
	docID := uuid.New()

	docKey := "plans/1.md"
	docType := "plan"
	title := "Plan"
	oldHash := "old"
	content := "new content"
	newHash := hashContent(content)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id, current_version
FROM mb_documents
WHERE project_id = $1 AND doc_key = $2
FOR UPDATE
`)).
		WithArgs(projectID, docKey).
		WillReturnRows(sqlmock.NewRows([]string{"id", "current_version"}).AddRow(docID, 2))

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT content_hash
FROM mb_document_versions
WHERE document_id = $1 AND version = $2
`)).
		WithArgs(docID, 2).
		WillReturnRows(sqlmock.NewRows([]string{"content_hash"}).AddRow(oldHash))

	// Insert new version (v=3)
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO mb_document_versions (id, document_id, version, content, content_hash)
VALUES ($1, $2, $3, $4, $5)
`)).
		WithArgs(sqlmock.AnyArg(), docID, 3, content, newHash).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Update mb_documents current_version=3
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE mb_documents
SET doc_type = $3, title = NULLIF($4, ''), current_version = $5, updated_at = now()
WHERE id = $1 AND project_id = $2
`)).
		WithArgs(docID, projectID, docType, title, 3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	out, err := r.UpsertDocument(context.Background(), projectID, docKey, docType, title, content)
	if err != nil {
		t.Fatalf("UpsertDocument: %v", err)
	}
	if out.Version != 3 {
		t.Fatalf("Version mismatch: got %d want %d", out.Version, 3)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

