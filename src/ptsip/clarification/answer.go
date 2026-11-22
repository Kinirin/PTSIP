// Package clarification reads the canonical clarification answer format.
package clarification

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/source"
)

const AnswerFormat = "ptsip-clarification-answer/v2"

var answerBlock = regexp.MustCompile("(?is)```(?:ya?ml)?\\s*\\n(.*?)```")

type Answer struct {
	Classification  string `json:"classification"`
	Purpose         string `json:"purpose"`
	Shipped         bool   `json:"shipped"`
	RuntimeRequired bool   `json:"runtime_required"`
	Executable      bool   `json:"executable"`
}

type Resolution struct {
	Valid  bool     `json:"valid"`
	Status string   `json:"status"`
	Errors []string `json:"errors"`
}

// ParseAnswer accepts standalone YAML or the latest matching YAML block in an
// issue answer. Parsed facts do not authorize a Profile change.
func ParseAnswer(text string) (Answer, error) {
	if len(text) > source.MaxBytes {
		return Answer{}, fmt.Errorf("clarification answer exceeds size limit")
	}
	matches := answerBlock.FindAllStringSubmatch(text, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		if document := answerDocument(matches[i][1]); document != nil {
			return decodeAnswer(document)
		}
	}
	if document := answerDocument(text); document != nil {
		return decodeAnswer(document)
	}
	return Answer{}, fmt.Errorf("no structured clarification answer with format %q", AnswerFormat)
}

func answerDocument(text string) source.Document {
	document, err := source.Decode([]byte(text))
	if err != nil || document["format"] != AnswerFormat {
		return nil
	}
	return document
}

func decodeAnswer(document source.Document) (Answer, error) {
	decision := source.Object(document["decision"])
	fields := []string{"classification", "purpose", "shipped", "runtime_required", "executable"}
	if len(decision) != len(fields) {
		return Answer{}, fmt.Errorf("%s decision fields must match the canonical shape exactly", AnswerFormat)
	}
	for _, field := range fields {
		if _, exists := decision[field]; !exists {
			return Answer{}, fmt.Errorf("%s decision is missing %q", AnswerFormat, field)
		}
	}
	classification, ok := decision["classification"].(string)
	if !ok {
		return Answer{}, fmt.Errorf("classification must be a string")
	}
	purpose, ok := decision["purpose"].(string)
	if !ok {
		return Answer{}, fmt.Errorf("purpose must be a string")
	}
	answer := Answer{Classification: strings.ToUpper(strings.TrimSpace(classification)), Purpose: strings.TrimSpace(purpose)}
	for _, item := range []struct {
		field string
		value *bool
	}{{"shipped", &answer.Shipped}, {"runtime_required", &answer.RuntimeRequired}, {"executable", &answer.Executable}} {
		switch value := decision[item.field].(type) {
		case bool:
			*item.value = value
			continue
		case string:
			switch strings.ToUpper(strings.TrimSpace(value)) {
			case "YES", "TRUE":
				*item.value = true
				continue
			case "NO", "FALSE":
				continue
			}
		}
		return Answer{}, fmt.Errorf("%s must be YES or NO", item.field)
	}
	return answer, nil
}

func (answer Answer) Validate() Resolution {
	result := Resolution{Valid: true, Status: "RESOLVED", Errors: []string{}}
	switch answer.Classification {
	case "PRODUCT":
	case "DEVELOPMENT_TOOLING", "DELIVERY", "OPERATIONS":
		if answer.Shipped {
			result.Errors = append(result.Errors, answer.Classification+" component cannot be declared as shipped Product implementation")
		}
		if answer.RuntimeRequired {
			result.Errors = append(result.Errors, answer.Classification+" component cannot be required as Product runtime implementation")
		}
	case "NEUTRAL_CONTRACT":
		if answer.Executable {
			result.Errors = append(result.Errors, "NEUTRAL_CONTRACT must be non-executable")
		}
	default:
		result.Errors = append(result.Errors, "classification must be PRODUCT, DEVELOPMENT_TOOLING, DELIVERY, OPERATIONS, or NEUTRAL_CONTRACT")
	}
	if answer.Purpose == "" {
		result.Errors = append(result.Errors, "purpose must be non-empty")
	}
	if len(result.Errors) > 0 {
		result.Valid = false
		result.Status = "CONFLICT"
	}
	return result
}
