package machine

import (
	"fmt"
	"strings"
)

func agentOption(options map[string]string, name, defaultValue string) string {
	value := options["--"+name]
	if value == "" {
		return defaultValue
	}
	return value
}
func agentCheckResult(errors []string) (Object, error) {
	state := "CURRENT"
	if len(errors) > 0 {
		state = "STALE"
	}
	result := Object{"state": state, "errors": errors}
	if len(errors) > 0 {
		return result, Fail("STALE_AGENT_PROJECTION", strings.Join(errors, "; "))
	}
	return result, nil
}
func init() {
	RegisterOperations("agent-instruction-classifier", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		if command != "classify" {
			return nil, fmt.Errorf("unsupported classifier command %s", command)
		}
		return ClassifyAgentFile(r, agentOption(options, "source", "AGENTS.md"))
	})
	RegisterOperations("agent-instruction-materializer", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		source, output := agentOption(options, "source", "AGENTS.md"), agentOption(options, "output", ".agent")
		switch command {
		case "status":
			files, err := BuildAgentMaterialization(r, source, output)
			if err != nil {
				return nil, err
			}
			outputRef, err := r.Scope(output)
			if err != nil {
				return nil, err
			}
			return Object{"state": "PREVIEW", "summary": files[outputRef+"/registry.yaml"]["summary"], "level_2_materialized": false}, nil
		case "materialize":
			return MaterializeAgent(r, source, output)
		case "check":
			result, err := CheckAgentMaterialization(r, source, output)
			if err != nil {
				return nil, err
			}
			if result["state"] != "CURRENT" {
				return result, Fail("STALE_AGENT_PROJECTION", "Level 1 materialization is stale")
			}
			return result, nil
		}
		return nil, fmt.Errorf("unsupported materializer command %s", command)
	})
	RegisterOperations("agent-instruction-entry", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		if command != "resolve" {
			return nil, fmt.Errorf("unsupported entry command %s", command)
		}
		include := []string{}
		if raw := options["--include"]; raw != "" {
			include = strings.Split(raw, ",")
		}
		return ResolveAgentEntry(r, options["--operation"], include, agentOption(options, "agent-root", ".agent"))
	})
	RegisterOperations("agent-instruction-progressive", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "bootstrap-level1", "migrate-level1":
			return MigrateAgentLevel1(r)
		case "check":
			errors, err := CheckAgentProgressive(r)
			if err != nil {
				return nil, err
			}
			return agentCheckResult(errors)
		}
		return nil, fmt.Errorf("unsupported progressive command %s", command)
	})
	RegisterOperations("agent-instruction-activation", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "activate":
			return MigrateAgentLevel1(r)
		case "check":
			errors, err := CheckAgentProgressive(r)
			if err != nil {
				return nil, err
			}
			return agentCheckResult(errors)
		}
		return nil, fmt.Errorf("unsupported activation command %s", command)
	})
	RegisterOperations("agent-integration", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		switch command {
		case "status":
			return AgentIntegrationStatus(r)
		case "install-mcp":
			return InstallAgentMCP(r, BoolOption(options, "user-approved"))
		}
		return nil, fmt.Errorf("unsupported integration command %s", command)
	})
	RegisterOperations("context-projection", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		return ContextProjectionOperation(r, command, options["--input"])
	})
	RegisterOperations("agent-context-migration", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		if command != "verify" {
			return nil, fmt.Errorf("unsupported migration command %s", command)
		}
		result, err := VerifyAgentContextMigration(r, agentOption(options, "stage", "AUTO"))
		if err != nil {
			return nil, err
		}
		if result["status"] != "PASS" {
			return result, Fail("AGENT_MIGRATION_VERIFICATION_FAILED", "agent context migration checks failed")
		}
		return result, nil
	})
	RegisterOperations("implementation-work-packet", func(r *Repository, command string, options map[string]string, args []string) (any, error) {
		writeResult := func(result Object) (any, error) {
			if output := options["--output"]; output != "" {
				if err := r.WriteJSON(output, result, nil); err != nil {
					return nil, err
				}
			}
			return result, nil
		}
		if command == "prepare" {
			packet, err := BuildWorkPacket(r, options["--scope"], options["--operation"])
			if err != nil {
				return nil, err
			}
			if output := options["--output"]; output != "" {
				if err := r.WriteJSON(output, packet, nil); err != nil {
					return nil, err
				}
			}
			brief, err := BuildWorkPacketBrief(packet)
			if err != nil {
				return nil, err
			}
			if output := options["--brief-output"]; output != "" {
				if err := r.WriteJSON(output, brief, nil); err != nil {
					return nil, err
				}
			}
			if BoolOption(options, "agent-brief") {
				return brief, nil
			}
			return packet, nil
		}
		packet, err := r.Read(options["--packet"])
		if err != nil {
			return nil, err
		}
		checked, err := CheckWorkPacket(r, packet)
		if err != nil {
			return nil, err
		}
		if command == "check" {
			if checked["status"] != "PASS" {
				return checked, Fail("PACKET_BLOCKED", "work packet is stale or out of scope")
			}
			return checked, nil
		}
		if checked["status"] != "PASS" {
			return checked, Fail("PACKET_BLOCKED", "work packet must be current before use")
		}
		switch command {
		case "brief":
			brief, err := BuildWorkPacketBrief(packet)
			if err != nil {
				return nil, err
			}
			return writeResult(brief)
		case "context":
			context, err := BuildWorkPacketSourceContext(r, packet, options["--role"], options["--context-id"])
			if err != nil {
				return nil, err
			}
			return writeResult(context)
		case "verify":
			return VerifyWorkPacket(r, packet, options["--stage"], options["--log"], options["--failure-state"])
		}
		return nil, fmt.Errorf("unsupported work packet command %s", command)
	})
}
