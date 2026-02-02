package service

import "context"

type ITxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
