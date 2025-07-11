package db

import (
	"fmt"
	"slices"
	"strings"
)

type ACLPolicy struct {
	Role     string `json:"role"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func (a *ACLPolicy) Validate() error {
	if a.Role == "" {
		return fmt.Errorf("role cannot be empty")
	}
	if a.Resource == "" {
		return fmt.Errorf("resource cannot be empty")
	}
	if a.Action == "" {
		return fmt.Errorf("action cannot be empty")
	}
	role := strings.ToLower(a.Role)
	if !slices.Contains(ValidUserRoles(), role) {
		return fmt.Errorf("role must be one of: %s", strings.Join(ValidUserRoles(), ", "))
	}
	action := strings.ToLower(a.Action)
	if action != "get" && action != "post" && action != "put" && action != "delete" && action != "head" && action != "patch" && action != "options" {
		return fmt.Errorf("action must be one of: get, post, put, delete, head, patch, options, *")
	}
	return nil
}

func SliceToACLPolicies(slice [][]string) []ACLPolicy {
	var policies []ACLPolicy
	for _, p := range slice {
		if len(p) != 3 {
			continue // Skip invalid entries
		}
		policies = append(policies, ACLPolicy{
			Role:     p[0],
			Resource: p[1],
			Action:   p[2],
		})
	}
	return policies
}
