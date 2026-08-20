param(
    [Parameter(Position = 0)]
    [ValidateSet("start-quick", "stop", "status")]
    [string]$Command = "status",

    [ValidateRange(1, 65535)]
    [int]$GatewayPort = 8766,

    [switch]$IncludeCapabilityUrl
)

$ErrorActionPreference = "Stop"

# Repository-local lifecycle entrypoint for the installed OpenAI Local Bridge runtime.
# This script MUST NOT resolve or execute another source checkout of
# Kinirin/OpenAI_Local_Bridge. Runtime/state lives under LOCALAPPDATA/PROGRAMDATA;
# the PTSIP repository owns this invocation surface so checkout location/branch
# changes in the Bridge source repository cannot break PTSIP verification commands.
$BridgeRoot = Join-Path $env:LOCALAPPDATA "OpenAI_Local_Bridge"
$VenvPython = Join-Path $BridgeRoot "venv\Scripts\python.exe"
$RemoteRoot = Join-Path $BridgeRoot "remote"
$StateFile = Join-Path $RemoteRoot "state.json"
$TokenFile = Join-Path $RemoteRoot "capability.token"
$GatewayLog = Join-Path $RemoteRoot "gateway.log"
$GatewayStdout = Join-Path $RemoteRoot "gateway.stdout.log"
$GatewayStderr = Join-Path $RemoteRoot "gateway.stderr.log"
$CloudflaredStdout = Join-Path $RemoteRoot "cloudflared.stdout.log"
$CloudflaredStderr = Join-Path $RemoteRoot "cloudflared.stderr.log"
$ActivationFile = Join-Path $env:PROGRAMDATA "OpenAI_Local_Bridge\activation.json"

function Test-IsAdministrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Assert-Administrator {
    if (-not (Test-IsAdministrator)) {
        throw "Remote transport lifecycle changes require an Administrator PowerShell session."
    }
}

function Resolve-Cloudflared {
    foreach ($name in @("cloudflared.exe", "cloudflared")) {
        $command = Get-Command $name -ErrorAction SilentlyContinue
        if ($command) {
            return $command.Source
        }
    }

    $candidates = @(
        (Join-Path $env:ProgramFiles "cloudflared\cloudflared.exe"),
        (Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Links\cloudflared.exe")
    )
    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }
    }

    throw "cloudflared was not found. Install Cloudflare Tunnel and ensure cloudflared.exe is available on PATH."
}

function Get-BridgeActivation {
    if (-not (Test-Path -LiteralPath $ActivationFile)) {
        throw "OpenAI Local Bridge host is not activated. Activate the installed Bridge host first."
    }
    return (Get-Content -LiteralPath $ActivationFile -Encoding UTF8 | Out-String | ConvertFrom-Json)
}

function Assert-BridgeReady {
    $activation = Get-BridgeActivation
    $uri = "http://127.0.0.1:$([int]$activation.port)/health"
    try {
        $health = Invoke-RestMethod -Method Get -Uri $uri -TimeoutSec 2
    } catch {
        throw "Activated Bridge is not reachable at $uri. Restart the installed base Bridge before enabling remote transport."
    }
    if ($health.ok -ne $true) {
        throw "Activated Bridge health check failed."
    }
    return $activation
}

function New-CapabilityToken {
    $bytes = New-Object byte[] 32
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $rng.GetBytes($bytes)
    } finally {
        $rng.Dispose()
    }
    return [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

function Write-SecretText {
    param([string]$Path, [string]$Value)
    $encoding = New-Object System.Text.UTF8Encoding($false)
    [IO.File]::WriteAllText($Path, $Value, $encoding)
}

function ConvertTo-ProcessId {
    param([object]$Value)
    if ($null -eq $Value) {
        return $null
    }
    $parsed = 0
    if (-not [int]::TryParse([string]$Value, [ref]$parsed)) {
        return $null
    }
    if ($parsed -le 0) {
        return $null
    }
    return $parsed
}

function Test-ProcessAlive {
    param([object]$ProcessId)
    $resolvedId = ConvertTo-ProcessId -Value $ProcessId
    if ($null -eq $resolvedId) {
        return $false
    }
    return [bool](Get-Process -Id $resolvedId -ErrorAction SilentlyContinue)
}

function Stop-TrackedProcess {
    param(
        [object]$ProcessId,
        [string]$ExpectedName,
        [string]$ExpectedExecutable = "",
        [string]$ExpectedCommandLinePattern = ""
    )
    $resolvedId = ConvertTo-ProcessId -Value $ProcessId
    if ($null -eq $resolvedId) {
        return
    }
    $process = Get-Process -Id $resolvedId -ErrorAction SilentlyContinue
    if (-not $process) {
        return
    }
    if ($ExpectedName -and $process.ProcessName -notlike $ExpectedName) {
        Write-Warning "Skipping stale PID ${resolvedId}: expected process '$ExpectedName', found '$($process.ProcessName)'."
        return
    }

    $processInfo = Get-CimInstance Win32_Process -Filter "ProcessId = $resolvedId" -ErrorAction SilentlyContinue
    if (-not $processInfo) {
        if (-not (Get-Process -Id $resolvedId -ErrorAction SilentlyContinue)) {
            return
        }
        throw "Refusing to stop PID ${resolvedId}: process identity could not be verified."
    }

    if (-not [string]::IsNullOrWhiteSpace($ExpectedExecutable)) {
        $actualExecutable = [string]$processInfo.ExecutablePath
        if ([string]::IsNullOrWhiteSpace($actualExecutable) -or
            -not [string]::Equals(
                [IO.Path]::GetFullPath($actualExecutable),
                [IO.Path]::GetFullPath($ExpectedExecutable),
                [StringComparison]::OrdinalIgnoreCase
            )) {
            Write-Warning "Skipping stale PID ${resolvedId}: executable identity does not match the tracked remote process."
            return
        }
    }

    if (-not [string]::IsNullOrWhiteSpace($ExpectedCommandLinePattern)) {
        $actualCommandLine = [string]$processInfo.CommandLine
        if ([string]::IsNullOrWhiteSpace($actualCommandLine) -or
            $actualCommandLine -notmatch $ExpectedCommandLinePattern) {
            Write-Warning "Skipping stale PID ${resolvedId}: command-line role does not match the tracked remote process."
            return
        }
    }
    Stop-Process -Id $resolvedId -Force
}

function Read-RemoteState {
    if (-not (Test-Path -LiteralPath $StateFile)) {
        return $null
    }
    return (Get-Content -LiteralPath $StateFile -Encoding UTF8 | Out-String | ConvertFrom-Json)
}

function Test-HttpHealth {
    param([string]$Uri)
    try {
        $response = Invoke-RestMethod -Method Get -Uri $Uri -TimeoutSec 5
        return ($response.ok -eq $true)
    } catch {
        return $false
    }
}

function Wait-ForLocalGateway {
    param([string]$HealthUri)
    for ($attempt = 1; $attempt -le 20; $attempt++) {
        if (Test-HttpHealth -Uri $HealthUri) {
            return
        }
        Start-Sleep -Milliseconds 500
    }
    throw "Remote gateway did not become ready at $HealthUri."
}

function Wait-ForQuickTunnelUrl {
    for ($attempt = 1; $attempt -le 60; $attempt++) {
        $text = ""
        foreach ($path in @($CloudflaredStdout, $CloudflaredStderr)) {
            if (Test-Path -LiteralPath $path) {
                $text += "`n" + (Get-Content -LiteralPath $path -Encoding UTF8 -ErrorAction SilentlyContinue | Out-String)
            }
        }
        $match = [regex]::Match($text, 'https://[a-z0-9-]+\.trycloudflare\.com', [Text.RegularExpressions.RegexOptions]::IgnoreCase)
        if ($match.Success) {
            return $match.Value.TrimEnd('/')
        }
        Start-Sleep -Milliseconds 500
    }
    throw "cloudflared did not report a TryCloudflare URL. Inspect $CloudflaredStderr."
}

function Remove-RemoteRuntimeFiles {
    foreach ($path in @($StateFile, $TokenFile)) {
        Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue
    }
}

function Stop-RemoteInternal {
    $state = Read-RemoteState
    if ($state) {
        $cloudflaredExecutable = ""
        if ($state.PSObject.Properties.Name -contains "cloudflared_executable") {
            $cloudflaredExecutable = [string]$state.cloudflared_executable
        }
        $cloudflaredPattern = '^\s*(?:"[^"]+"|\S+)\s+tunnel\s+--url\s+http://127\.0\.0\.1:' +
            [regex]::Escape([string]$state.gateway_port) + '(?:\s|$)'
        $gatewayPattern = '^\s*(?:"[^"]+"|\S+)\s+-m\s+openai_local_bridge\.remote_gateway(?:\s|$)'

        Stop-TrackedProcess `
            -ProcessId $state.cloudflared_pid `
            -ExpectedName "cloudflared*" `
            -ExpectedExecutable $cloudflaredExecutable `
            -ExpectedCommandLinePattern $cloudflaredPattern
        Stop-TrackedProcess `
            -ProcessId $state.gateway_pid `
            -ExpectedName "python*" `
            -ExpectedExecutable $VenvPython `
            -ExpectedCommandLinePattern $gatewayPattern
    }
    Remove-RemoteRuntimeFiles
}

function Get-RemoteStatus {
    $state = Read-RemoteState
    if (-not $state) {
        return [PSCustomObject][ordered]@{
            active = $false
            provider = $null
            mode = $null
            public_mcp_url = $null
            gateway_running = $false
            tunnel_running = $false
            local_gateway_reachable = $false
            public_reachable = $false
        }
    }

    $gatewayRunning = Test-ProcessAlive -ProcessId $state.gateway_pid
    $tunnelRunning = Test-ProcessAlive -ProcessId $state.cloudflared_pid
    $localHealth = $false
    $publicHealth = $false
    if ($gatewayRunning) {
        $localHealth = Test-HttpHealth -Uri ([string]$state.local_health_url)
    }
    if ($tunnelRunning) {
        $publicHealth = Test-HttpHealth -Uri ([string]$state.public_health_url)
    }

    return [PSCustomObject][ordered]@{
        active = [bool]($gatewayRunning -and $tunnelRunning -and $localHealth -and $publicHealth)
        provider = [string]$state.provider
        mode = [string]$state.mode
        public_mcp_url = [string]$state.public_mcp_url
        gateway_running = [bool]$gatewayRunning
        tunnel_running = [bool]$tunnelRunning
        local_gateway_reachable = [bool]$localHealth
        public_reachable = [bool]$publicHealth
    }
}

function Write-RemoteStatus {
    param([object]$Status)
    $output = [ordered]@{
        active = [bool]$Status.active
        provider = $Status.provider
        mode = $Status.mode
        public_mcp_url = if ($IncludeCapabilityUrl) { [string]$Status.public_mcp_url } else { $null }
        gateway_running = [bool]$Status.gateway_running
        tunnel_running = [bool]$Status.tunnel_running
        local_gateway_reachable = [bool]$Status.local_gateway_reachable
        public_reachable = [bool]$Status.public_reachable
        capability_url_exposed = [bool]$IncludeCapabilityUrl
    }
    [PSCustomObject]$output | ConvertTo-Json -Depth 5
}

switch ($Command) {
    "start-quick" {
        Assert-Administrator
        if (-not (Test-Path -LiteralPath $VenvPython)) {
            throw "Bridge runtime is not installed at $VenvPython. Activate/install the base Bridge runtime first."
        }
        $activation = Assert-BridgeReady
        $cloudflared = Resolve-Cloudflared
        Stop-RemoteInternal
        $activation = Assert-BridgeReady
        New-Item -ItemType Directory -Force -Path $RemoteRoot | Out-Null

        $token = New-CapabilityToken
        Write-SecretText -Path $TokenFile -Value $token
        $localHealthUrl = "http://127.0.0.1:$GatewayPort/$token/health"

        $gatewayArgs = @(
            "-m", "openai_local_bridge.remote_gateway",
            "--listen-port", "$GatewayPort",
            "--bridge-port", "$([int]$activation.port)",
            "--token-file", "`"$TokenFile`"",
            "--log-file", "`"$GatewayLog`""
        )
        $gatewayProcess = Start-Process `
            -FilePath $VenvPython `
            -ArgumentList $gatewayArgs `
            -PassThru `
            -WindowStyle Hidden `
            -RedirectStandardOutput $GatewayStdout `
            -RedirectStandardError $GatewayStderr

        try {
            Wait-ForLocalGateway -HealthUri $localHealthUrl

            Remove-Item -LiteralPath $CloudflaredStdout, $CloudflaredStderr -Force -ErrorAction SilentlyContinue
            $cloudflaredProcess = Start-Process `
                -FilePath $cloudflared `
                -ArgumentList @("tunnel", "--url", "http://127.0.0.1:$GatewayPort") `
                -PassThru `
                -WindowStyle Hidden `
                -RedirectStandardOutput $CloudflaredStdout `
                -RedirectStandardError $CloudflaredStderr

            try {
                $quickBaseUrl = Wait-ForQuickTunnelUrl
                $publicHealthUrl = "$quickBaseUrl/$token/health"
                $publicMcpUrl = "$quickBaseUrl/$token/mcp"

                for ($attempt = 1; $attempt -le 20; $attempt++) {
                    if (Test-HttpHealth -Uri $publicHealthUrl) {
                        break
                    }
                    Start-Sleep -Milliseconds 500
                }
                if (-not (Test-HttpHealth -Uri $publicHealthUrl)) {
                    throw "TryCloudflare URL was created but the remote gateway is not reachable."
                }

                $state = [ordered]@{
                    schema = "openai-local-bridge-remote/v1"
                    provider = "cloudflare"
                    mode = "quick"
                    gateway_port = $GatewayPort
                    bridge_port = [int]$activation.port
                    gateway_pid = [int]$gatewayProcess.Id
                    cloudflared_pid = [int]$cloudflaredProcess.Id
                    cloudflared_executable = [string]$cloudflaredProcess.Path
                    local_health_url = $localHealthUrl
                    public_health_url = $publicHealthUrl
                    public_mcp_url = $publicMcpUrl
                    started_at = [DateTime]::UtcNow.ToString("o")
                }
                $state | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $StateFile -Encoding UTF8

                $status = Get-RemoteStatus
                if (-not $status.active) {
                    Stop-RemoteInternal
                    throw "Remote transport startup completed but final process/health verification failed. Inspect logs under $RemoteRoot."
                }
                Write-RemoteStatus -Status $status
            } catch {
                if ($cloudflaredProcess) {
                    Stop-Process -Id $cloudflaredProcess.Id -Force -ErrorAction SilentlyContinue
                }
                throw
            }
        } catch {
            Stop-Process -Id $gatewayProcess.Id -Force -ErrorAction SilentlyContinue
            Remove-RemoteRuntimeFiles
            throw
        }
        break
    }
    "stop" {
        Assert-Administrator
        Stop-RemoteInternal
        Write-Host "OpenAI Local Bridge remote transport is stopped."
        break
    }
    "status" {
        $status = Get-RemoteStatus
        Write-RemoteStatus -Status $status
        if (-not $status.active) {
            exit 6
        }
        break
    }
}
