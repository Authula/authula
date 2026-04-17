package tests

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

type MockTxRunner struct {
	run func(func(context.Context, bun.Tx) error) error
}

func (r *MockTxRunner) RunInTx(ctx context.Context, _ *sql.TxOptions, fn func(context.Context, bun.Tx) error) error {
	if r.run != nil {
		return r.run(fn)
	}
	var tx bun.Tx
	return fn(ctx, tx)
}

type MockOrganizationInvitationTxRunner struct {
	run func(func(context.Context, bun.Tx) error) error
}

func (r *MockOrganizationInvitationTxRunner) RunInTx(ctx context.Context, _ *sql.TxOptions, fn func(context.Context, bun.Tx) error) error {
	if r.run != nil {
		return r.run(fn)
	}
	var tx bun.Tx
	return fn(ctx, tx)
}
