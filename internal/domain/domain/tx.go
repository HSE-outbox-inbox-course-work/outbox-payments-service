package domain

import "context"

type Tx interface {
	Commit(context.Context) error
	Rollback(context.Context) error
}
