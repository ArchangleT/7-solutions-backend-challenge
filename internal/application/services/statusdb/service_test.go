package statusdb

import (
	"context"
	"errors"
	"testing"

	"github.com/7-solutions/backend-challenge/internal/mocks/statusdb"
	"go.uber.org/mock/gomock"
)

func TestCount(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := statusdbmocks.NewMockRepository(ctrl)
	repo.EXPECT().Count(gomock.Any()).Return(int64(2), nil)

	svc := NewService(repo)
	n, err := svc.Count(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("Count() = %d, want 2", n)
	}
}

func TestCount_Error(t *testing.T) {
	ctrl := gomock.NewController(t)

	wantErr := errors.New("boom")
	repo := statusdbmocks.NewMockRepository(ctrl)
	repo.EXPECT().Count(gomock.Any()).Return(int64(0), wantErr)

	svc := NewService(repo)
	_, err := svc.Count(context.Background())
	if !errors.Is(err, wantErr) {
		t.Errorf("Count() error = %v, want %v", err, wantErr)
	}
}
