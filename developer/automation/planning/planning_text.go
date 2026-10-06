package planning

import (
	"fmt"
	"regexp"
	"strings"
)

func planningSectionSpan(text, section string) (int, int, error) {
	marker := regexp.MustCompile("(?m)^" + regexp.QuoteMeta(section) + ":[ \\t]*$")
	start := marker.FindStringIndex(text)
	if start == nil {
		return 0, 0, fmt.Errorf("top-level section not found: %s", section)
	}
	next := regexp.MustCompile("(?m)^[A-Za-z0-9_][A-Za-z0-9_-]*:").FindStringIndex(text[start[1]:])
	end := len(text)
	if next != nil {
		end = start[1] + next[0]
	}
	return start[0], end, nil
}
func planningStageSpan(text, id string) (int, int, string, error) {
	marker := regexp.MustCompile("(?m)^([ \\t]*)- id: " + regexp.QuoteMeta(id) + "[ \\t]*$")
	match := marker.FindStringSubmatchIndex(text)
	if match == nil {
		return 0, 0, "", fmt.Errorf("stage marker missing: %s", id)
	}
	if len(marker.FindAllStringIndex(text, -1)) != 1 {
		return 0, 0, "", fmt.Errorf("stage marker ambiguous: %s", id)
	}
	indent := text[match[2]:match[3]]
	next := regexp.MustCompile("(?m)^" + regexp.QuoteMeta(indent) + "- id: ").FindStringIndex(text[match[1]:])
	end := len(text)
	if next != nil {
		end = match[1] + next[0]
	}
	return match[0], end, indent, nil
}
func planningReplaceStageStatus(text, id, old, next string) (string, error) {
	start, end, indent, err := planningStageSpan(text, id)
	if err != nil {
		return "", err
	}
	block := text[start:end]
	status := regexp.MustCompile("(?m)^" + regexp.QuoteMeta(indent) + "  status: " + regexp.QuoteMeta(old) + "[ \\t]*$")
	if len(status.FindAllStringIndex(block, -1)) != 1 {
		return "", fmt.Errorf("stage status does not match exactly once: %s", id)
	}
	block = status.ReplaceAllString(block, indent+"  status: "+next)
	return text[:start] + block + text[end:], nil
}
func PlanningPromoteStageText(text, id string, automatic Object) (string, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	result, err := planningReplaceStageStatus(text, id, "IMPLEMENTED_VALIDATION_PENDING", "COMPLETE")
	if err != nil {
		return "", err
	}
	start, end, indent, err := planningStageSpan(result, id)
	if err != nil {
		return "", err
	}
	block := result[start:end]
	validation := regexp.MustCompile("(?m)^(" + regexp.QuoteMeta(indent) + "  validation:[ \\t]*\n" + regexp.QuoteMeta(indent) + "    status:) PENDING[ \\t]*$")
	block = validation.ReplaceAllString(block, "${1} PASS")
	result = result[:start] + block + result[end:]
	if raw := automatic["next_stage"]; raw != nil {
		next := Map(raw)
		if next == nil {
			return "", fmt.Errorf("next_stage must be mapping")
		}
		nextID, old, to := Text(next["id"]), Text(next["from_status"]), Text(next["to_status"])
		if nextID != "" && old != "" && to != "" {
			result, err = planningReplaceStageStatus(result, nextID, old, to)
			if err != nil {
				return "", err
			}
		}
	}
	updates := Map(automatic["document_updates"])
	if automatic["document_updates"] != nil && updates == nil {
		return "", fmt.Errorf("document_updates must be mapping")
	}
	for _, kind := range []string{"mapping_scalars", "list_removals"} {
		if raw := updates[kind]; raw != nil && List(raw) == nil {
			return "", fmt.Errorf("document_updates.%s must be list", kind)
		}
	}
	for _, raw := range List(updates["mapping_scalars"]) {
		row := Map(raw)
		section, key, field, old, to := Text(row["section"]), Text(row["key"]), Text(row["field"]), Text(row["from_value"]), Text(row["to_value"])
		if section == "" || key == "" || field == "" || old == "" || to == "" {
			return "", fmt.Errorf("mapping scalar update requires nonempty strings")
		}
		sectionStart, sectionEnd, err := planningSectionSpan(result, section)
		if err != nil {
			return "", err
		}
		sectionText := result[sectionStart:sectionEnd]
		keyMarker := regexp.MustCompile("(?m)^  " + regexp.QuoteMeta(key) + ":[ \\t]*$")
		keyMatch := keyMarker.FindStringIndex(sectionText)
		if keyMatch == nil {
			return "", fmt.Errorf("mapping key missing: %s.%s", section, key)
		}
		nextKey := regexp.MustCompile("(?m)^  [A-Za-z0-9_][A-Za-z0-9_-]*:").FindStringIndex(sectionText[keyMatch[1]:])
		keyEnd := len(sectionText)
		if nextKey != nil {
			keyEnd = keyMatch[1] + nextKey[0]
		}
		block := sectionText[keyMatch[0]:keyEnd]
		pattern := regexp.MustCompile("(?m)^    " + regexp.QuoteMeta(field) + ": " + regexp.QuoteMeta(old) + "[ \\t]*$")
		if len(pattern.FindAllStringIndex(block, -1)) != 1 {
			return "", fmt.Errorf("mapping scalar mismatch: %s.%s.%s", section, key, field)
		}
		block = pattern.ReplaceAllString(block, "    "+field+": "+to)
		sectionText = sectionText[:keyMatch[0]] + block + sectionText[keyEnd:]
		result = result[:sectionStart] + sectionText + result[sectionEnd:]
	}
	for _, raw := range List(updates["list_removals"]) {
		row := Map(raw)
		section, value := Text(row["section"]), Text(row["value"])
		if section == "" || value == "" {
			return "", fmt.Errorf("list removal requires section and value")
		}
		start, end, err := planningSectionSpan(result, section)
		if err != nil {
			return "", err
		}
		block := result[start:end]
		pattern := regexp.MustCompile("(?m)^  - " + regexp.QuoteMeta(value) + "[ \\t]*(?:\n|$)")
		if len(pattern.FindAllStringIndex(block, -1)) != 1 {
			return "", fmt.Errorf("list item mismatch: %s -> %s", section, value)
		}
		block = pattern.ReplaceAllString(block, "")
		parts := strings.SplitN(block, "\n", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) == "" {
			block = section + ": []\n\n"
		}
		result = result[:start] + block + result[end:]
	}
	return result, nil
}
func planningReplaceExtensionStatuses(text string, payload Object) (string, error) {
	result := strings.ReplaceAll(text, "\r\n", "\n")
	for _, definition := range []struct{ section, pattern, old, next string }{{"extension", "(?m)^(  lifecycle:[ \\t]*\n    status:) ACTIVE[ \\t]*$", "ACTIVE", "COMPLETE"}, {"implementation_authorization", "(?m)^(  status:) AUTHORIZED[ \\t]*$", "AUTHORIZED", "COMPLETE"}} {
		status := ""
		if definition.section == "extension" {
			status = planningStatus(Map(payload["extension"]))
		} else {
			status = Text(Map(payload[definition.section])["status"])
		}
		if status == definition.next {
			continue
		}
		if status != definition.old {
			return "", fmt.Errorf("extension status mismatch")
		}
		start, end, err := planningSectionSpan(result, definition.section)
		if err != nil {
			return "", err
		}
		block := result[start:end]
		pattern := regexp.MustCompile(definition.pattern)
		if len(pattern.FindAllStringIndex(block, -1)) != 1 {
			return "", fmt.Errorf("extension status text unresolved")
		}
		block = pattern.ReplaceAllString(block, "${1} "+definition.next)
		result = result[:start] + block + result[end:]
	}
	return result, nil
}
