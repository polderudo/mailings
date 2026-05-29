package db

import (
	"context"
	"fmt"
	"strings"
)

type UserProfileTableRow struct {
	ID            string `json:"id"`
	FullName      string `json:"fullName"`
	Email         string `json:"email"`
	AccountStatus string `json:"accountStatus"`
}

func NormalizeUserProfileTableFilterRequest(request FilterRequest[map[string]string]) FilterRequest[map[string]string] {
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
			Field:  "fullName",
			IsDesc: false,
		}}
	}
	return request
}

func buildUserProfileTableWhereClause(filters map[string]string) (string, []any) {
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

	addLikeTwo := func(pattern string, value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		args = append(args, Like(trimmed))
		index := len(args)
		clauses = append(clauses, fmt.Sprintf(pattern, index, index))
	}

	addLikeOne("CAST(id AS TEXT) ILIKE $%d", filters["id"])
	addLikeTwo("(CONCAT_WS(' ', forename, lastname) ILIKE $%d OR email ILIKE $%d)", filters["fullName"])
	addLikeOne("(CASE WHEN disabled THEN 'failed' ELSE 'success' END) ILIKE $%d", filters["accountStatus"])

	if v := strings.TrimSpace(filters["filterStatus"]); v != "" {
		if v == "active" {
			clauses = append(clauses, "NOT up.disabled")
		} else if v == "disabled" {
			clauses = append(clauses, "up.disabled")
		}
	}

	if len(clauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildUserProfileTableOrderByClause(sorts []SortParam) string {
	if len(sorts) == 0 {
		return " ORDER BY forename ASC, lastname ASC"
	}

	clauses := make([]string, 0, len(sorts)+1)
	for _, sort := range sorts {
		direction := "ASC"
		if sort.IsDesc {
			direction = "DESC"
		}

		switch sort.Field {
		case "id":
			clauses = append(clauses, "id "+direction)
		case "fullName":
			clauses = append(clauses, "forename "+direction, "lastname "+direction)
		case "accountStatus":
			clauses = append(clauses, "disabled "+direction)
		case "createdAt":
			clauses = append(clauses, "created_at "+direction)
		}
	}

	if len(clauses) == 0 {
		return " ORDER BY forename ASC, lastname ASC"
	}

	return " ORDER BY " + strings.Join(clauses, ", ")
}

func LoadUserProfileTableRows(ctx context.Context, request FilterRequest[map[string]string]) ([]UserProfileTableRow, int64, error) {
	request = NormalizeUserProfileTableFilterRequest(request)

	whereClause, whereArgs := buildUserProfileTableWhereClause(request.FilterCriteria)

	var total int64
	countQuery := `SELECT COUNT(*) FROM user_profile up` + whereClause
	if err := StdDB.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append([]any(nil), whereArgs...)
	queryArgs = append(queryArgs, request.P.Count, request.P.Offset())

	rows, err := StdDB.QueryContext(
		ctx,
		`SELECT up.id, up.email, up.forename, up.lastname, up.disabled
FROM user_profile up`+
			whereClause+
			buildUserProfileTableOrderByClause(request.P.Sorts)+
			fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(queryArgs)-1, len(queryArgs)),
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := make([]UserProfileTableRow, 0, request.P.Count)
	for rows.Next() {
		var (
			id       int32
			email    string
			forename string
			lastname string
			disabled bool
		)

		if err := rows.Scan(
			&id,
			&email,
			&forename,
			&lastname,
			&disabled,
		); err != nil {
			return nil, 0, err
		}

		fullName := strings.TrimSpace(strings.Join([]string{forename, lastname}, " "))
		if fullName == "" {
			fullName = email
		}

		accountStatus := "active"
		if disabled {
			accountStatus = "disabled"
		}

		result = append(result, UserProfileTableRow{
			ID:            fmt.Sprintf("%d", id),
			FullName:      fullName,
			Email:         email,
			AccountStatus: accountStatus,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, total, nil
}
