package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type MailTemplateTableRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	UpdatedAt string `json:"updatedAt"`
}

func NormalizeMailTemplateTableFilterRequest(request FilterRequest[map[string]string]) FilterRequest[map[string]string] {
	if request.FilterCriteria == nil {
		request.FilterCriteria = map[string]string{}
	}
	if request.P.Page < 1 {
		request.P.Page = 1
	}
	if request.P.Count <= 0 {
		request.P.Count = 50
	}
	if len(request.P.Sorts) == 0 {
		request.P.Sorts = []SortParam{{
			Field:  "name",
			IsDesc: false,
		}}
	}
	return request
}

func buildMailTemplateTableWhereClause(filters map[string]string) (string, []any) {
	clauses := make([]string, 0, len(filters))
	args := make([]any, 0, len(filters))

	addLikeOne := func(pattern string, value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		args = append(args, Like(trimmed))
		index := len(args)
		clauses = append(clauses, fmt.Sprintf(pattern, index))
	}

	addLikeOne("CAST(id AS TEXT) ILIKE $%d", filters["id"])
	addLikeOne("name ILIKE $%d", filters["name"])
	addLikeOne("subject ILIKE $%d", filters["subject"])

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildMailTemplateTableOrderByClause(sorts []SortParam) string {
	if len(sorts) == 0 {
		return " ORDER BY name ASC"
	}
	clauses := make([]string, 0, len(sorts))
	for _, sort := range sorts {
		direction := "ASC"
		if sort.IsDesc {
			direction = "DESC"
		}
		switch sort.Field {
		case "id":
			clauses = append(clauses, "id "+direction)
		case "name":
			clauses = append(clauses, "name "+direction)
		case "subject":
			clauses = append(clauses, "subject "+direction)
		case "updatedAt":
			clauses = append(clauses, "COALESCE(updated_at, created_at) "+direction)
		}
	}
	if len(clauses) == 0 {
		return " ORDER BY name ASC"
	}
	return " ORDER BY " + strings.Join(clauses, ", ")
}

func LoadMailTemplateTableRows(ctx context.Context, request FilterRequest[map[string]string]) ([]MailTemplateTableRow, int64, error) {
	request = NormalizeMailTemplateTableFilterRequest(request)

	whereClause, whereArgs := buildMailTemplateTableWhereClause(request.FilterCriteria)

	var total int64
	countQuery := `SELECT COUNT(*) FROM mail_template` + whereClause
	if err := StdDB.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append([]any(nil), whereArgs...)
	queryArgs = append(queryArgs, request.P.Count, request.P.Offset())

	rows, err := StdDB.QueryContext(
		ctx,
		`SELECT id, name, subject, COALESCE(updated_at, created_at)
FROM mail_template`+
			whereClause+
			buildMailTemplateTableOrderByClause(request.P.Sorts)+
			fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(queryArgs)-1, len(queryArgs)),
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := make([]MailTemplateTableRow, 0, request.P.Count)
	for rows.Next() {
		var (
			id        int32
			name      string
			subject   string
			updatedAt time.Time
		)
		if err := rows.Scan(&id, &name, &subject, &updatedAt); err != nil {
			return nil, 0, err
		}
		result = append(result, MailTemplateTableRow{
			ID:        fmt.Sprintf("%d", id),
			Name:      name,
			Subject:   subject,
			UpdatedAt: updatedAt.Format("2006-01-02 15:04"),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}
