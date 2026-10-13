[CmdletBinding()]
param(
    [ValidateSet('Select','DispatchTest','DispatchRelease','ResolvePackage','Install','Identities')]
    [string]$Action = 'Select',
    [string]$Targets = 'package',
    [string]$OptionalPlatforms = '',
    [ValidateSet('github','github-pypi')][string]$Destination = 'github-pypi',
    [string]$PackageVersion = '',
    [string]$PythonExecutable = '',
    [string]$PackageRegistry = '.github/checks/release/package-registry.json'
)
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../..'))
$controller = Join-Path $PSScriptRoot 'controller'
switch ($Action) {
    'DispatchTest' {
        & gh workflow run tooling-test.yml --repo Kinirin/PTSIP --ref main -f "targets=$Targets" -f "optional_platforms=$OptionalPlatforms"
    }
    'DispatchRelease' {
        & gh workflow run tooling-release.yml --repo Kinirin/PTSIP --ref main -f "targets=$Targets" -f "optional_platforms=$OptionalPlatforms" -f "destination=$Destination"
    }
    default {
        $command = switch ($Action) {
            'Select' { 'select' }
            'ResolvePackage' { 'resolve-package' }
            'Install' { 'install-package' }
            'Identities' { 'identities' }
        }
        $arguments = @('-C',$controller,'run','./cmd/ptsip-release',$command,'--root',$repo)
        if ($Action -eq 'Select') { $arguments += @('--targets',$Targets,'--optional-platforms',$OptionalPlatforms) }
        else {
            if (-not $PackageVersion) { throw 'PackageVersion must be a numeric base or an exact base.SHA6 identity.' }
            $arguments += @('--version',$PackageVersion,'--registry',$PackageRegistry)
            if ($Action -eq 'Install') {
                if (-not $PythonExecutable) { throw 'An explicit existing target Python environment is required.' }
                $arguments += @('--python',$PythonExecutable)
            }
        }
        & go @arguments
    }
}
if ($LASTEXITCODE -ne 0) { throw "Registered release action failed: $Action" }
