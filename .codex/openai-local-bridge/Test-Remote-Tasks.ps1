param(
    [string]$RepositoryId = "Kinirin/PTSIP",
    [Parameter(Mandatory = $true)]
    [string]$TaskId
)

$ErrorActionPreference = "Stop"
$ProtocolVersion = "2025-11-25"
$RemoteScript = Join-Path $PSScriptRoot "OpenAI-Local-Bridge-Remote.ps1"
$script:NextRequestId = 10

# Repository-local task acceptance runner.
# It intentionally has no dependency on a Kinirin/OpenAI_Local_Bridge source
# checkout or that checkout's virtual environments. It consumes only the
# installed Bridge runtime/remote state through the sibling lifecycle script.

function ConvertTo-CompactJson {
    param([hashtable]$Payload)
    return ($Payload | ConvertTo-Json -Depth 12 -Compress)
}

function Invoke-McpPost {
    param(
        [string]$Uri,
        [hashtable]$Payload,
        [string]$SessionId = ""
    )

    $headers = @{
        Accept = "application/json, text/event-stream"
        Origin = "https://chatgpt.com"
    }
    if (-not [string]::IsNullOrWhiteSpace($SessionId)) {
        $headers["MCP-Session-Id"] = $SessionId
        $headers["MCP-Protocol-Version"] = $ProtocolVersion
    }

    return Invoke-WebRequest `
        -UseBasicParsing `
        -Method Post `
        -Uri $Uri `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (ConvertTo-CompactJson -Payload $Payload) `
        -TimeoutSec 30
}

function Invoke-McpTool {
    param(
        [string]$Uri,
        [string]$SessionId,
        [string]$Name,
        [hashtable]$Arguments
    )

    $script:NextRequestId += 1
    $response = Invoke-McpPost `
        -Uri $Uri `
        -SessionId $SessionId `
        -Payload @{
            jsonrpc = "2.0"
            id = $script:NextRequestId
            method = "tools/call"
            params = @{
                name = $Name
                arguments = $Arguments
            }
        }
    $body = $response.Content | ConvertFrom-Json
    if ($body.error) {
        throw "MCP tool '$Name' failed: $($body.error.message)"
    }
    if ($body.result.isError -eq $true) {
        $detail = @($body.result.content | ForEach-Object { $_.text }) -join "`n"
        throw "MCP tool '$Name' returned an error: $detail"
    }
    return $body.result.structuredContent
}

if (-not (Test-Path -LiteralPath $RemoteScript)) {
    throw "Repository-local Bridge lifecycle entrypoint is missing: $RemoteScript"
}

$statusText = (& $RemoteScript status -IncludeCapabilityUrl | Out-String).Trim()
$statusExitCode = $LASTEXITCODE
if ([string]::IsNullOrWhiteSpace($statusText)) {
    throw "Remote transport status did not return JSON."
}
try {
    $remoteStatus = $statusText | ConvertFrom-Json
} catch {
    throw "Remote transport status returned invalid JSON."
}
if ($statusExitCode -ne 0 -or $remoteStatus.active -ne $true -or $remoteStatus.public_reachable -ne $true) {
    throw (
        "Remote transport is not healthy: active={0}, gateway_running={1}, tunnel_running={2}, " +
        "local_gateway_reachable={3}, public_reachable={4}. " +
        "Use .\.codex\openai-local-bridge\OpenAI-Local-Bridge-Remote.ps1 start-quick from an Administrator PowerShell session."
    ) -f @(
        $remoteStatus.active,
        $remoteStatus.gateway_running,
        $remoteStatus.tunnel_running,
        $remoteStatus.local_gateway_reachable,
        $remoteStatus.public_reachable
    )
}
$mcpUri = [string]$remoteStatus.public_mcp_url
if ([string]::IsNullOrWhiteSpace($mcpUri)) {
    throw "Remote transport did not provide an internal MCP capability URL."
}

$sessionId = ""
$finalState = $null
$log = $null
try {
    $initialize = Invoke-McpPost `
        -Uri $mcpUri `
        -Payload @{
            jsonrpc = "2.0"
            id = 1
            method = "initialize"
            params = @{
                protocolVersion = $ProtocolVersion
                capabilities = @{}
                clientInfo = @{
                    name = "ptsip-repository-local-remote-task-acceptance"
                    version = "1.0"
                }
            }
        }
    $initializeBody = $initialize.Content | ConvertFrom-Json
    $sessionId = [string]$initialize.Headers["MCP-Session-Id"]
    if ([string]::IsNullOrWhiteSpace($sessionId)) {
        throw "Remote initialize did not return MCP-Session-Id."
    }
    if ($initializeBody.result.protocolVersion -ne $ProtocolVersion) {
        throw "Unexpected MCP protocol version: $($initializeBody.result.protocolVersion)"
    }

    $notification = Invoke-McpPost `
        -Uri $mcpUri `
        -SessionId $sessionId `
        -Payload @{
            jsonrpc = "2.0"
            method = "notifications/initialized"
            params = @{}
        }
    if ([int]$notification.StatusCode -ne 202) {
        throw "notifications/initialized expected HTTP 202."
    }

    $toolsResponse = Invoke-McpPost `
        -Uri $mcpUri `
        -SessionId $sessionId `
        -Payload @{
            jsonrpc = "2.0"
            id = 2
            method = "tools/list"
            params = @{}
        }
    $toolsBody = $toolsResponse.Content | ConvertFrom-Json
    $toolNames = @($toolsBody.result.tools | ForEach-Object { [string]$_.name })
    foreach ($required in @(
        "local_repositories",
        "local_tasks",
        "local_run_task",
        "local_run_status",
        "local_read_log"
    )) {
        if ($toolNames -notcontains $required) {
            throw "Required remote task tool is missing: $required"
        }
    }

    $repositories = Invoke-McpTool `
        -Uri $mcpUri `
        -SessionId $sessionId `
        -Name "local_repositories" `
        -Arguments @{}
    $repository = @($repositories.repositories | Where-Object {
        [string]$_.repository_id -ieq $RepositoryId
    }) | Select-Object -First 1
    if (-not $repository) {
        throw "Repository was not discovered through remote MCP: $RepositoryId"
    }

    $tasks = Invoke-McpTool `
        -Uri $mcpUri `
        -SessionId $sessionId `
        -Name "local_tasks" `
        -Arguments @{ repository_id = $RepositoryId }
    $task = @($tasks.tasks | Where-Object {
        [string]$_.task_id -eq $TaskId
    }) | Select-Object -First 1
    if (-not $task) {
        throw "Task was not registered: $RepositoryId/$TaskId"
    }
    if ([string]$task.access -ne "read") {
        throw "Remote acceptance task must be registered as read: $RepositoryId/$TaskId"
    }

    $started = Invoke-McpTool `
        -Uri $mcpUri `
        -SessionId $sessionId `
        -Name "local_run_task" `
        -Arguments @{
            repository_id = $RepositoryId
            task_id = $TaskId
        }
    $runId = [string]$started.run_id
    if ([string]::IsNullOrWhiteSpace($runId)) {
        throw "local_run_task did not return run_id."
    }

    for ($attempt = 1; $attempt -le 660; $attempt++) {
        $finalState = Invoke-McpTool `
            -Uri $mcpUri `
            -SessionId $sessionId `
            -Name "local_run_status" `
            -Arguments @{ run_id = $runId }
        if ([string]$finalState.status -ne "running") {
            break
        }
        Start-Sleep -Milliseconds 500
    }
    if (-not $finalState -or [string]$finalState.status -eq "running") {
        throw "Registered task did not reach a terminal state within the acceptance window."
    }

    $log = Invoke-McpTool `
        -Uri $mcpUri `
        -SessionId $sessionId `
        -Name "local_read_log" `
        -Arguments @{ run_id = $runId }
} finally {
    if (-not [string]::IsNullOrWhiteSpace($sessionId)) {
        $deleteHeaders = @{
            Origin = "https://chatgpt.com"
            "MCP-Session-Id" = $sessionId
            "MCP-Protocol-Version" = $ProtocolVersion
        }
        try {
            Invoke-WebRequest `
                -UseBasicParsing `
                -Method Delete `
                -Uri $mcpUri `
                -Headers $deleteHeaders `
                -TimeoutSec 15 | Out-Null
        } catch {
        }
    }
}

$passed = ($finalState.status -eq "completed" -and [int]$finalState.exit_code -eq 0)
[PSCustomObject][ordered]@{
    passed = [bool]$passed
    repository_id = $RepositoryId
    task_id = $TaskId
    run_id = [string]$finalState.run_id
    status = [string]$finalState.status
    exit_code = $finalState.exit_code
    failure_kind = $finalState.failure_kind
    protocol_version = $ProtocolVersion
    remote_task_execution_verified = [bool]$passed
    log_available = [bool]$log.available
    log_truncated = [bool]$log.truncated
    capability_url_exposed = $false
} | ConvertTo-Json -Depth 6

if (-not $passed) {
    exit 7
}
