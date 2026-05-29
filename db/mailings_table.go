package db

import (
	"context"
	"fmt"
	"strings"
)

type MailingTableRow struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	TemplateName   string `json:"templateName"`
	ListName       string `json:"listName"`
	Domain         string `json:"domain"`
	Status         string `json:"status"`
	RecipientCount string `json:"recipientCount"`
	SentCount      string `json:"sentCount"`
}

func NormalizeMailingTableFilterRequest(request FilterRequest[map[string]string]) FilterRequest[map[string]string] {
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
		request.P.Sorts = []SortParam{{Field: "createdAt", IsDesc: true}}
	}
	return request
}

func buildMailingTableWhereClause(filters map[string]string) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)

	addLikeOne := func(pattern string, value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		args = append(args, Like(trimmed))
		idx := len(args)
		clauses = append(clauses, fmt.Sprintf(pattern, idx))
	}

	addLikeOne("CAST(m.id AS TEXT) ILIKE $%d", filters["id"])
	addLikeOne("m.name ILIKE $%d", filters["name"])
	addLikeOne("m.status ILIKE $%d", filters["status"])

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildMailingTableOrderByClause(sorts []SortParam) string {
	if len(sorts) == 0 {
		return " ORDER BY m.created_at DESC"
	}
	clauses := make([]string, 0, len(sorts))
	for _, s := range sorts {
		dir := "ASC"
		if s.IsDesc {
			dir = "DESC"
		}
		switch s.Field {
		case "id":
			clauses = append(clauses, "m.id "+dir)
		case "name":
			clauses = append(clauses, "m.name "+dir)
		case "status":
			clauses = append(clauses, "m.status "+dir)
		case "createdAt":
			clauses = append(clauses, "m.created_at "+dir)
		}
	}
	if len(clauses) == 0 {
		return " ORDER BY m.created_at DESC"
	}
	return " ORDER BY " + strings.Join(clauses, ", ")
}

func LoadMailingTableRows(ctx context.Context, request FilterRequest[map[string]string]) ([]MailingTableRow, int64, error) {
	request = NormalizeMailingTableFilterRequest(request)

	whereClause, whereArgs := buildMailingTableWhereClause(request.FilterCriteria)

	var total int64
	countQuery := `SELECT COUNT(*) FROM mailing m` + whereClause
	if err := StdDB.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append([]any(nil), whereArgs...)
	queryArgs = append(queryArgs, request.P.Count, request.P.Offset())

	rows, err := StdDB.QueryContext(
		ctx,
		`SELECT m.id, m.name, t.name, l.name, d.domain, m.status,
       (SELECT COUNT(*) FROM mail_list_recipient r WHERE r.list_id = m.list_id) AS recipient_count,
       m.sent_count
FROM mailing m
JOIN mail_template t ON t.id = m.template_id
JOIN mail_list l     ON l.id = m.list_id
JOIN mail_domain d   ON d.id = m.domain_id`+
			whereClause+
			buildMailingTableOrderByClause(request.P.Sorts)+
			fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(queryArgs)-1, len(queryArgs)),
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := make([]MailingTableRow, 0, request.P.Count)
	for rows.Next() {
		var (
			id       int32
			name     string
			tName    string
			lName    string
			domain   string
			status   string
			recCount int64
			sent     int32
		)
		if err := rows.Scan(&id, &name, &tName, &lName, &domain, &status, &recCount, &sent); err != nil {
			return nil, 0, err
		}
		result = append(result, MailingTableRow{
			ID:             fmt.Sprintf("%d", id),
			Name:           name,
			TemplateName:   tName,
			ListName:       lName,
			Domain:         domain,
			Status:         status,
			RecipientCount: fmt.Sprintf("%d", recCount),
			SentCount:      fmt.Sprintf("%d", sent),
		})
	}
	return result, total, rows.Err()
}
