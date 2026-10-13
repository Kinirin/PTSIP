[CmdletBinding()]
param(
    [ValidateSet('Select','DispatchTest','DispatchRelease')][string]$Action='Select',
    [string]$VerificationRunId='',
    [ValidateSet('github','github-pypi')][string]$Destination='github-pypi',
    [string]$OptionalPlatforms=''
)
$ErrorActionPreference='Stop'
& (Join-Path $PSScriptRoot '../run_release.ps1') -Action $Action -Targets 'engine:storage' -VerificationRunId $VerificationRunId -Destination $Destination -OptionalPlatforms $OptionalPlatforms
