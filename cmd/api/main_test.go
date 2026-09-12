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

func TestCountUsersLoop(t *testing.T) {
	errBoom := errors.New("boom")
	testInterval := 10 * time.Millisecond

	tests := []struct {
		name        string
		counts      []int64
		countErrors []error
		ctxSetup    func() (context.Context, context.CancelFunc)
		wantLogs    []string
	}{
		{
			name:        "logs count every tick",
			counts:      []int64{0, 5, 10},
			countErrors: []error{nil, nil, nil},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 50*time.Millisecond)
			},
			wantLogs: []string{"user count task started", "total users", "user count task stopped"},
		},
		{
			name:        "logs error and continues on repo failure",
			counts:      []int64{0, 0},
			countErrors: []error{errBoom, nil},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 50*time.Millisecond)
			},
			wantLogs: []string{"user count task started", "failed to count users", "total users", "user count task stopped"},
		},
		{
			name:        "exits immediately when context already cancelled",
			counts:      []int64{},
			countErrors: []error{},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantLogs: []string{"user count task started", "user count task stopped"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := statusdbmocks.NewMockRepository(ctrl)

			for i := range tt.counts {
				repo.EXPECT().Count(gomock.Any()).Return(tt.counts[i], tt.countErrors[i]).Times(1)
			}

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