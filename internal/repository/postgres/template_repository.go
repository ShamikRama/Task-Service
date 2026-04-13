package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
	tpldomain "example.com/taskservice/internal/domain/tasktemplate"
	appcache "example.com/taskservice/internal/infrastructure/cache"
	infrapg "example.com/taskservice/internal/infrastructure/postgres"
	tplusecase "example.com/taskservice/internal/usecase/tasktemplate"
)

type dbConn interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type TemplateRepository struct {
	pool     *pgxpool.Pool
	txMgr    *infrapg.TransactionManager
	tplCache appcache.KeyValueCache[int64, *tpldomain.Template]
}

func NewTemplateRepository(pool *pgxpool.Pool, txMgr *infrapg.TransactionManager, tplCache appcache.KeyValueCache[int64, *tpldomain.Template]) *TemplateRepository {
	return &TemplateRepository{pool: pool, txMgr: txMgr, tplCache: tplCache}
}

func NewTemplateMapCache() appcache.KeyValueCache[int64, *tpldomain.Template] {
	return appcache.NewMapCache[int64, *tpldomain.Template]()
}

func (r *TemplateRepository) CreateWithTasks(ctx context.Context, tpl *tpldomain.Template, tasks []taskdomain.Task) (*tpldomain.Template, int64, error) {
	var out *tpldomain.Template
	var inserted int64
	err := r.txMgr.Run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		created, err := r.create(ctx, tx, tpl)
		if err != nil {
			return err
		}
		out = created
		if len(tasks) == 0 {
			return nil
		}
		tid := created.ID
		execTasks := make([]taskdomain.Task, len(tasks))
		for i := range tasks {
			execTasks[i] = tasks[i]
			execTasks[i].TemplateID = &tid
		}
		n, err := r.batchCreateTasks(ctx, tx, execTasks)
		inserted = n
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	r.cacheSet(ctx, out)
	return out, inserted, nil
}

func (r *TemplateRepository) create(ctx context.Context, db dbConn, t *tpldomain.Template) (*tpldomain.Template, error) {
	const query = `
		INSERT INTO task_templates (title, description, periodicity_type, periodicity_params, start_date, end_date, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, periodicity_type, periodicity_params, start_date, end_date, is_active, created_at, updated_at
	`

	row := db.QueryRow(ctx, query,
		t.Title, t.Description, t.PeriodicityType, t.RawPeriodicityParams,
		t.StartDate, t.EndDate, t.IsActive, t.CreatedAt, t.UpdatedAt,
	)

	return scanTemplate(row)
}

func (r *TemplateRepository) GetByID(ctx context.Context, id int64) (*tpldomain.Template, error) {
	if r.tplCache != nil {
		if v, ok := r.tplCache.Get(ctx, id); ok && v != nil {
			cp, err := cloneTemplate(v)
			if err == nil {
				return cp, nil
			}
		}
	}

	const query = `
		SELECT id, title, description, periodicity_type, periodicity_params, start_date, end_date, is_active, created_at, updated_at
		FROM task_templates
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	tpl, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, tpldomain.ErrNotFound
		}
		return nil, err
	}

	r.cacheSet(ctx, tpl)
	return tpl, nil
}

func (r *TemplateRepository) Update(ctx context.Context, t *tpldomain.Template) (*tpldomain.Template, error) {
	const query = `
		UPDATE task_templates
		SET title = $1,
			description = $2,
			periodicity_type = $3,
			periodicity_params = $4,
			start_date = $5,
			end_date = $6,
			is_active = $7,
			updated_at = $8
		WHERE id = $9
		RETURNING id, title, description, periodicity_type, periodicity_params, start_date, end_date, is_active, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query,
		t.Title, t.Description, t.PeriodicityType, t.RawPeriodicityParams,
		t.StartDate, t.EndDate, t.IsActive, t.UpdatedAt, t.ID,
	)

	updated, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, tpldomain.ErrNotFound
		}
		return nil, err
	}

	r.cacheSet(ctx, updated)
	return updated, nil
}

func (r *TemplateRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM task_templates WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return tpldomain.ErrNotFound
	}

	r.cacheDelete(ctx, id)
	return nil
}

func (r *TemplateRepository) cacheSet(ctx context.Context, tpl *tpldomain.Template) {
	if r.tplCache == nil || tpl == nil {
		return
	}
	cp, err := cloneTemplate(tpl)
	if err != nil {
		return
	}
	r.tplCache.Set(ctx, tpl.ID, cp)
}

func (r *TemplateRepository) cacheDelete(ctx context.Context, id int64) {
	if r.tplCache == nil {
		return
	}
	r.tplCache.Delete(ctx, id)
}

func (r *TemplateRepository) List(ctx context.Context, filter tplusecase.ListFilter) ([]tpldomain.Template, error) {
	query := `
		SELECT id, title, description, periodicity_type, periodicity_params, start_date, end_date, is_active, created_at, updated_at
		FROM task_templates
	`

	if filter.ActiveOnly {
		query += ` WHERE is_active = TRUE`
	}

	query += ` ORDER BY id DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]tpldomain.Template, 0)
	for rows.Next() {
		tpl, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *tpl)
	}

	return templates, rows.Err()
}

func (r *TemplateRepository) BatchCreateTasks(ctx context.Context, tasks []taskdomain.Task) (int64, error) {
	return r.batchCreateTasks(ctx, r.pool, tasks)
}

func (r *TemplateRepository) batchCreateTasks(ctx context.Context, db dbConn, tasks []taskdomain.Task) (int64, error) {
	if len(tasks) == 0 {
		return 0, nil
	}

	var (
		sb     strings.Builder
		args   []any
		argIdx int
	)

	sb.WriteString(`INSERT INTO tasks (title, description, status, template_id, scheduled_date, created_at, updated_at) VALUES `)

	for i, t := range tasks {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6, argIdx+7))

		args = append(args, t.Title, t.Description, t.Status, t.TemplateID, t.ScheduledDate, t.CreatedAt, t.UpdatedAt)
		argIdx += 7
	}

	sb.WriteString(` ON CONFLICT (template_id, scheduled_date) WHERE template_id IS NOT NULL DO NOTHING`)

	result, err := db.Exec(ctx, sb.String(), args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

type templateScanner interface {
	Scan(dest ...any) error
}

func cloneTemplate(t *tpldomain.Template) (*tpldomain.Template, error) {
	if t == nil {
		return nil, nil
	}
	cp := *t
	cp.RawPeriodicityParams = append(json.RawMessage(nil), t.RawPeriodicityParams...)
	if t.EndDate != nil {
		ed := *t.EndDate
		cp.EndDate = &ed
	}
	params, err := tpldomain.ParsePeriodicityParams(cp.PeriodicityType, cp.RawPeriodicityParams)
	if err != nil {
		return nil, err
	}
	cp.PeriodicityParams = params
	return &cp, nil
}

func scanTemplate(scanner templateScanner) (*tpldomain.Template, error) {
	var (
		t         tpldomain.Template
		ptype     string
		rawParams json.RawMessage
		startDate time.Time
		endDate   *time.Time
	)

	if err := scanner.Scan(
		&t.ID,
		&t.Title,
		&t.Description,
		&ptype,
		&rawParams,
		&startDate,
		&endDate,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		return nil, err
	}

	t.PeriodicityType = tpldomain.PeriodicityType(ptype)
	t.StartDate = startDate
	t.EndDate = endDate
	t.RawPeriodicityParams = rawParams

	params, err := tpldomain.ParsePeriodicityParams(t.PeriodicityType, rawParams)
	if err != nil {
		return nil, fmt.Errorf("parse periodicity params for template %d: %w", t.ID, err)
	}
	t.PeriodicityParams = params

	return &t, nil
}
