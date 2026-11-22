package contracts_test

import (
	"strings"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/clarification"
)

func TestGoClarificationAnswerV2(t *testing.T) {
	const document = "format: ptsip-clarification-answer/v2\ndecision:\n" +
		"  classification: product\n  purpose: '  사용자 기능  '\n" +
		"  shipped: YES\n  runtime_required: 'no'\n  executable: true\n"
	const fence = "```"
	input := "Reply\n" + fence + "yaml\n" + strings.Replace(document, "product", "operations", 1) + fence +
		"\nCorrection\n" + fence + "yml\n" + document + fence
	answer, err := clarification.ParseAnswer(input)
	if err != nil || answer.Classification != "PRODUCT" || answer.Purpose != "사용자 기능" ||
		!answer.Shipped || answer.RuntimeRequired || !answer.Executable {
		t.Fatalf("unexpected answer: %#v, %v", answer, err)
	}
	if result := answer.Validate(); !result.Valid || result.Status != "RESOLVED" || len(result.Errors) != 0 {
		t.Fatalf("product answer failed: %#v", result)
	}
	answer.Classification, answer.RuntimeRequired = "DELIVERY", true
	result := answer.Validate()
	if result.Valid || result.Status != "CONFLICT" || len(result.Errors) != 2 {
		t.Fatalf("invalid Product declarations accepted: %#v", result)
	}
	answer.Classification = "NEUTRAL_CONTRACT"
	result = answer.Validate()
	if result.Valid || len(result.Errors) != 1 || result.Errors[0] != "NEUTRAL_CONTRACT must be non-executable" {
		t.Fatalf("executable neutral contract accepted: %#v", result)
	}

	for _, bad := range []string{
		strings.Replace(document, "answer/v2", "answer/v1", 1),
		document + "  lifecycle_owner: PRODUCT\n",
		document + "  shipped: NO\n",
		strings.Replace(document, "shipped: YES", "shipped: maybe", 1),
	} {
		if _, err := clarification.ParseAnswer(bad); err == nil {
			t.Errorf("accepted an invalid v2 answer: %q", bad)
		}
	}
}
