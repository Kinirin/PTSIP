package binding

import (
	"sort"
	"strings"
)

// Store owns Policy-Plan relationship resolution, mutation and reconciliation.
// The repository adapter supplies mechanical reads, paths and schema validation.
type Store struct{ Repository }

func NewStore(repo Repository) *Store { return &Store{Repository: repo} }

func createdRelation(row Object) string {
	sections := append([]string(nil), Strings(row["policy_sections"])...)
	sort.Strings(sections)
	return Text(row["policy_ref"]) + "\x00" + strings.Join(sections, "\x00") + "\x00" + Text(row["resolved_plan_id"])
}
