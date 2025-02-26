// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package db

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.createNodeStmt, err = db.PrepareContext(ctx, createNode); err != nil {
		return nil, fmt.Errorf("error preparing query CreateNode: %w", err)
	}
	if q.createUserStmt, err = db.PrepareContext(ctx, createUser); err != nil {
		return nil, fmt.Errorf("error preparing query CreateUser: %w", err)
	}
	if q.deleteNodeStmt, err = db.PrepareContext(ctx, deleteNode); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteNode: %w", err)
	}
	if q.findUserByEmailStmt, err = db.PrepareContext(ctx, findUserByEmail); err != nil {
		return nil, fmt.Errorf("error preparing query FindUserByEmail: %w", err)
	}
	if q.findUserByIdStmt, err = db.PrepareContext(ctx, findUserById); err != nil {
		return nil, fmt.Errorf("error preparing query FindUserById: %w", err)
	}
	if q.getNodeStmt, err = db.PrepareContext(ctx, getNode); err != nil {
		return nil, fmt.Errorf("error preparing query GetNode: %w", err)
	}
	if q.getNodesStmt, err = db.PrepareContext(ctx, getNodes); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodes: %w", err)
	}
	if q.updateNodeStmt, err = db.PrepareContext(ctx, updateNode); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateNode: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.createNodeStmt != nil {
		if cerr := q.createNodeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createNodeStmt: %w", cerr)
		}
	}
	if q.createUserStmt != nil {
		if cerr := q.createUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createUserStmt: %w", cerr)
		}
	}
	if q.deleteNodeStmt != nil {
		if cerr := q.deleteNodeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteNodeStmt: %w", cerr)
		}
	}
	if q.findUserByEmailStmt != nil {
		if cerr := q.findUserByEmailStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing findUserByEmailStmt: %w", cerr)
		}
	}
	if q.findUserByIdStmt != nil {
		if cerr := q.findUserByIdStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing findUserByIdStmt: %w", cerr)
		}
	}
	if q.getNodeStmt != nil {
		if cerr := q.getNodeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodeStmt: %w", cerr)
		}
	}
	if q.getNodesStmt != nil {
		if cerr := q.getNodesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodesStmt: %w", cerr)
		}
	}
	if q.updateNodeStmt != nil {
		if cerr := q.updateNodeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateNodeStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                  DBTX
	tx                  *sql.Tx
	createNodeStmt      *sql.Stmt
	createUserStmt      *sql.Stmt
	deleteNodeStmt      *sql.Stmt
	findUserByEmailStmt *sql.Stmt
	findUserByIdStmt    *sql.Stmt
	getNodeStmt         *sql.Stmt
	getNodesStmt        *sql.Stmt
	updateNodeStmt      *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                  tx,
		tx:                  tx,
		createNodeStmt:      q.createNodeStmt,
		createUserStmt:      q.createUserStmt,
		deleteNodeStmt:      q.deleteNodeStmt,
		findUserByEmailStmt: q.findUserByEmailStmt,
		findUserByIdStmt:    q.findUserByIdStmt,
		getNodeStmt:         q.getNodeStmt,
		getNodesStmt:        q.getNodesStmt,
		updateNodeStmt:      q.updateNodeStmt,
	}
}
