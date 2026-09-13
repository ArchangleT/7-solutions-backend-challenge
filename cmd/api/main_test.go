package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/7-solutions/backend-challenge/internal/application/services/statusdb"
	statusdbmocks "github.com/7-solutions/backend-challenge/internal/mocks/statusdb"
	"go.uber.org/mock/gomock"
)

func TestRedactURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{name: "no credentials", uri: "mongodb://localhost:27017", want: "mongodb://localhost:27017"},
		{name: "credentials redacted", uri: "mongodb://user:pass@db.example.com:27017", want: "mongodb://***@db.example.com:27017"},
		{name: "no scheme", uri: "localhost:27017", want: "localhost:27017"},
		{name: "at sign in password redacted", uri: "mongodb+srv://u:p@ss@db.example.com", want: "mongodb+srv://***@db.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactURI(tt.uri); got != tt.want {
				t.Errorf("redactURI(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestCountUsersLoop(t *testing.T) {
	errBoom := errors.New("boom")
	testInterval := 10 * time.Millisecond

	tests := []struct {
		name      string
		setupMock func(*statusdbmocks.MockRepository)
		ctxSetup  func() (context.Context, context.CancelFunc)
	}{
		{
			name: "logs count every tick",
			setupMock: func(repo *statusdbmocks.MockRepository) {
				repo.EXPECT().Count(gomock.Any()).Return(int64(0), nil).AnyTimes()
				repo.EXPECT().Count(gomock.Any()).Return(int64(5), nil).AnyTimes()
				repo.EXPECT().Count(gomock.Any()).Return(int64(10), nil).AnyTimes()
			},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 50*time.Millisecond)
			},
		},
		{
			name: "logs error and continues on repo failure",
			setupMock: func(repo *statusdbmocks.MockRepository) {
				repo.EXPECT().Count(gomock.Any()).Return(int64(0), errBoom).AnyTimes()
				repo.EXPECT().Count(gomock.Any()).Return(int64(0), nil).AnyTimes()
			},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 50*time.Millisecond)
			},
		},
		{
			name: "exits immediately when context already cancelled",
			setupMock: func(repo *statusdbmocks.MockRepository) {
				// No calls expected
			},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := statusdbmocks.NewMockRepository(ctrl)

			tt.setupMock(repo)

			svc := statusdb.NewService(repo)

			ctx, cancel := tt.ctxSetup()
			defer cancel()

			done := make(chan struct{})
			go func() {
				countUsersLoop(ctx, svc, slog.Default(), testInterval)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("countUsersLoop did not exit in time")
			}
		})
	}
}
