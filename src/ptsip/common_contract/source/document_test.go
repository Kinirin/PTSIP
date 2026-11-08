package source

import (
	"os"
	"reflect"
	"testing"

	"github.com/Kinirin/PTSIP/src/ptsip/common_contract/jsondoc"
)

func TestCanonicalSourceGolden(t *testing.T) {
	raw, err := os.ReadFile("../testdata/canonical_source.json")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jsondoc.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	fixture := Object(decoded)
	expected := String(fixture["sha256"])
	for _, original := range []any{fixture["original"], yamlOriginal(t)} {
		got, err := Digest(original)
		if err != nil || got != expected {
			t.Fatalf("source digest = %q, %v; want %q", got, err, expected)
		}
		document := Document{
			"schema_version": Version,
			"Policy_Content": Document{
				"source_document": Document{"sha256": expected},
				"items":           []any{Document{"id": "SRC-0001", "source_pointer": "", "value": original}},
			},
		}
		restored, err := Restore(document)
		if err != nil || !reflect.DeepEqual(restored, original) {
			t.Fatalf("restored source = %#v, %v", restored, err)
		}
	}
}

func yamlOriginal(t *testing.T) Document {
	t.Helper()
	doc, err := Decode([]byte(`unicode: "\uC815\uCC45 \U0001F600"
decimal: 1.0
negative_zero: -0.0
small: 1.0e-6
large_int: 9007199254740993
nested: {"a/b": {"~": [null, true, 1]}}
`))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}
