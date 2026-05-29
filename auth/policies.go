package auth

import (
	"fmt"
)

func InitPolicies() error {
	policies := [][]string{
		{GroupOperator, OpMailing, ActionUpsert},
		{GroupOffice, OpMailing, ActionUpsert},
		{GroupDeveloper, OpDeveloper, ActionUpsert},
	}

	if _, err := enforcer.AddPolicies(policies); err != nil {
		return fmt.Errorf("error add policy: %w", err)
	}

	return nil
}

func RemoveGroupPolicies(group string) error {
	policy, err := enforcer.GetPolicy()
	if err != nil {
		return fmt.Errorf("error get policy: %w", err)
	}
	for _, p := range policy {
		if p[0] == group {
			if _, err := enforcer.RemovePolicy(p[0], p[1], p[2]); err != nil {
				return fmt.Errorf("error remove policy: %w", err)
			}
		}
	}

	return nil
}
