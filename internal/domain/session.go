package domain

import "context"

type SessionCapturer interface {
	SessionExists() bool
	CaptureSession(ctx context.Context) error
}
