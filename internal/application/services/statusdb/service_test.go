package statusdb

import (
	"context"
	"errors"
	"testing"

	statusdbmocks "github.com/ArchangleT/7-solutions-backend-challenge/internal/mocks/statusdb"
	"go.uber.org/mock/gomock"
)

func TestCount(t *testing.T) {
	errBoom := errors.New("boom")

	tests := []struct {
		name    string
		wantN   int64
		repoErr error
	}{
		{name: "returns repo count", wantN: 2},
		{name: "returns zero when empty", wantN: 0},
		{name: "propagates repo error", repoErr: errBoom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := statusdbmocks.NewMockRepository(ctrl)
			repo.EXPECT().Count(gomock.Any()).Return(tt.wantN, tt.repoErr)

			svc := NewService(repo)
			n, err := svc.Count(context.Background())

			if tt.repoErr != nil {
				if !errors.Is(err, tt.repoErr) {
					t.Fatalf("Count() error = %v, want %v", err, tt.repoErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Count() error = %v", err)
			}
			if n != tt.wantN {
				t.Errorf("Count() = %d, want %d", n, tt.wantN)
			}
		})
	}
}
