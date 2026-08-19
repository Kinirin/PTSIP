param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9a-fA-F]{32}$')]
    [string]$RunId,

    [string]$RepositoryId = "Kinirin/PTSIP",

    [ValidateRange(1000, 100000)]
    [int]$TailChars = 12000
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
    throw "LOCALAPPDATA is required to locate OpenAI Local Bridge run evidence."
}

if ($RepositoryId -notmatch '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$') {
    throw "RepositoryId must use owner/repository form."
}

$owner, $repo = $RepositoryId.Split('/', 2)
$runsRoot = Join-Path $env:LOCALAPPDATA "OpenAI_Local_Bridge\runs"
$evidenceDir = Join-Path (Join-Path (Join-Path $runsRoot $owner) $repo) $RunId

if (-not (Test-Path -LiteralPath $evidenceDir -PathType Container)) {
    throw "OpenAI Local Bridge evidence was not found for $RepositoryId run $RunId at $evidenceDir"
}

function Read-JsonFile {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return $null
    }
    $text = [System.IO.File]::ReadAllText($Path)
    if ([string]::IsNullOrWhiteSpace($text)) {
        return $null
    }
    return $text | ConvertFrom-Json
}

function Read-LogTail {
    param(
        [string]$Path,
        [int]$MaxChars
    )
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return [PSCustomObject]@{
            available = $false
            truncated = $false
            text = ""
        }
    }

    $text = [System.IO.File]::ReadAllText($Path)
    $truncated = $text.Length -gt $MaxChars
    if ($truncated) {
        $text = $text.Substring($text.Length - $MaxChars)
    }
    return [PSCustomObject]@{
        available = $true
        truncated = $truncated
        text = $text
    }
}

$request = Read-JsonFile (Join-Path $evidenceDir "request.json")
$result = Read-JsonFile (Join-Path $evidenceDir "result.json")
$stdout = Read-LogTail (Join-Path $evidenceDir "stdout.log") $TailChars
$stderr = Read-LogTail (Join-Path $evidenceDir "stderr.log") $TailChars

$gitHead = $null
$gitBranch = $null
$gitDirty = $null
if ($null -ne $request -and $null -ne $request.git) {
    $gitHead = $request.git.head_sha
    $gitBranch = $request.git.branch
    $gitDirty = $request.git.dirty
}

[PSCustomObject][ordered]@{
    run_id = $RunId
    repository_id = if ($null -ne $request -and $request.repository_id) { [string]$request.repository_id } else { $RepositoryId }
    task_id = if ($null -ne $request) { $request.task_id } else { $null }
    status = if ($null -ne $result) { $result.status } else { "UNKNOWN" }
    exit_code = if ($null -ne $result) { $result.exit_code } else { $null }
    failure_kind = if ($null -ne $result) { $result.failure_kind } else { $null }
    started_at = if ($null -ne $request) { $request.started_at } else { $null }
    finished_at = if ($null -ne $result) { $result.finished_at } else { $null }
    git_head_sha = $gitHead
    git_branch = $gitBranch
    git_dirty = $gitDirty
    log_available = [bool]($stdout.available -or $stderr.available)
    stdout_truncated = [bool]$stdout.truncated
    stderr_truncated = [bool]$stderr.truncated
    stdout_tail = [string]$stdout.text
    stderr_tail = [string]$stderr.text
} | ConvertTo-Json -Depth 8
