package policy

import (
	"crypto/sha256"
	"fmt"
	"regexp"
)

type Object = map[string]any

const DeveloperClass = "PTSIP_DEVELOPER_POLICY"

var RootID = regexp.MustCompile(`^MPD-(NORM|GOV|INTENT|ARCH|INFO|CNTR|RISK|SUPPLY|REAL|ASSURE|CTRL|CHANGE|OPS|RECORD)-[0-9]{4}$`)

type OperationError struct{ Code, Message string }

func (e *OperationError) Error() string { return e.Code + ": " + e.Message }

func Map(value any) Object  { object, _ := value.(map[string]any); return object }
func Text(value any) string { text, _ := value.(string); return text }
func List(value any) []any  { list, _ := value.([]any); return list }

func Strings(value any) []string {
	result := []string{}
	for _, item := range List(value) {
		result = append(result, Text(item))
	}
	return result
}

func SHA256(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
