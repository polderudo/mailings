package srvCtx

import (
	"app/db"
	"app/ilog"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/stephenafamo/bob"
)

type BobContext struct {
	Log         *slog.Logger
	EchoContext echo.Context
	Ctx         context.Context
	bobExecutor bob.Executor
	tx          bob.Transaction
}

func NewEchoBobCtx(c echo.Context) *BobContext {
	ret := &BobContext{
		Log:         ilog.NewEchoContextLogger(c),
		EchoContext: c,
		Ctx:         context.Background(),
		bobExecutor: db.DBob,
	}

	return ret
}

func NewLogBobCtx(l *slog.Logger) *BobContext {
	if l == nil {
		l = slog.Default()
	}

	return &BobContext{
		Log:         l,
		Ctx:         context.Background(),
		bobExecutor: db.DBob,
	}
}

func (m *BobContext) DropTX() {
	m.tx = nil
}

func (m *BobContext) DbTx() bob.Executor {
	if m.tx == nil {
		return m.bobExecutor
	}
	return m.tx
}

func (m *BobContext) StartTx() error {
	if m.tx != nil {
		return nil
	}

	var err error
	m.tx, err = db.DBob.Begin(m.Ctx)
	if err != nil {
		return fmt.Errorf("error starting bob transaction: %w", err)
	}
	m.bobExecutor = m.tx

	return nil
}

func (m *BobContext) CommitAndRestartTx() error {
	var err error
	if m.tx == nil {
		return nil
	}

	if err = m.tx.Commit(m.Ctx); err != nil {
		return fmt.Errorf("error commiting bob transaction: %w", err)
	}
	m.tx = nil

	if err := m.StartTx(); err != nil {
		return fmt.Errorf("error restarting bob transaction: %w", err)
	}

	return nil
}

func (m *BobContext) RollbackTx() error {
	var err error
	if m.tx != nil {
		if err = m.tx.Rollback(m.Ctx); err != nil {
			return fmt.Errorf("error rollback bob transaction: %w", err)
		}
	}

	return nil
}

func (m *BobContext) WithTx(fn func() error) error {
	txCreate := false
	if m.tx == nil {
		var err error
		txCreate = true
		if m.tx, err = db.DBob.Begin(m.Ctx); err != nil {
			return fmt.Errorf("error starting bob transaction: %w", err)
		}

		defer func() {
			if txCreate && m.tx != nil { //might be null, if already commited
				if err := m.tx.Rollback(m.Ctx); err != nil {
					if !errors.Is(err, sql.ErrTxDone) {
						m.Log.Error("error rolling back bob transaction", ilog.Err(err))
					}
				}
				m.tx = nil
			}
		}()
	}

	err := fn()
	if err == nil {
		//if no error, we commit the transaction if created
		//else any defer function in the stack will roll back the transaction
		if txCreate {
			if err := m.tx.Commit(m.Ctx); err != nil {
				return fmt.Errorf("error commiting transaction: %w", err)
			}
			m.tx = nil
		}
	}

	return err
}
