package db

import (
	"context"
	"fmt"
	"strings"
)

type MailListTableRow struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RecipientCount string `json:"recipientCount"`
}

func NormalizeMailListTableFilterRequest(request FilterRequest[map[string]string]) FilterRequest[map[string]string] {
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
		request.P.Sorts = []SortParam{{Field: "name", IsDesc: false}}
	}
	return request
}

func buildMailListTableWhereClause(filters map[string]string) (string, []any) {
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
	addLikeOne("description ILIKE $%d", filters["description"])

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildMailListTableOrderByClause(sorts []SortParam) string {
	if len(sorts) == 0 {
		return " ORDER BY ml.name ASC"
	}
	clauses := make([]string, 0, len(sorts))
	for _, sort := range sorts {
		direction := "ASC"
		if sort.IsDesc {
			direction = "DESC"
		}
		switch sort.Field {
		case "id":
			clauses = append(clauses, "ml.id "+direction)
		case "name":
			clauses = append(clauses, "ml.name "+direction)
		case "recipientCount":
			clauses = append(clauses, "recipient_count "+direction)
		}
	}
	if len(clauses) == 0 {
		return " ORDER BY ml.name ASC"
	}
	return " ORDER BY " + strings.Join(clauses, ", ")
}

func LoadMailListTableRows(ctx context.Context, request FilterRequest[map[string]string]) ([]MailListTableRow, int64, error) {
	request = NormalizeMailListTableFilterRequest(request)

	whereClause, whereArgs := buildMailListTableWhereClause(request.FilterCriteria)

	var total int64
	countQuery := `SELECT COUNT(*) FROM mail_list ml` + whereClause
	if err := StdDB.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append([]any(nil), whereArgs...)
	queryArgs = append(queryArgs, request.P.Count, request.P.Offset())

	rows, err := StdDB.QueryContext(
		ctx,
		`SELECT ml.id, ml.name, ml.description,
            (SELECT COUNT(*) FROM mail_list_recipient r WHERE r.list_id = ml.id) AS recipient_count
FROM mail_list ml`+
			whereClause+
			buildMailListTableOrderByClause(request.P.Sorts)+
			fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(queryArgs)-1, len(queryArgs)),
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := make([]MailListTableRow, 0, request.P.Count)
	for rows.Next() {
		var (
			id          int32
			name        string
			description string
			count       int64
		)
		if err := rows.Scan(&id, &name, &description, &count); err != nil {
			return nil, 0, err
		}
		result = append(result, MailListTableRow{
			ID:             fmt.Sprintf("%d", id),
			Name:           name,
			Description:    description,
			RecipientCount: fmt.Sprintf("%d", count),
		})
	}
	return result, total, rows.Err()
}
