[CmdletBinding()]
param(
    [ValidateSet('Select','DispatchTest','DispatchRelease')][string]$Action='Select',
    [ValidateSet('github','github-pypi')][string]$Destination='github-pypi',
    [string]$OptionalPlatforms=''
)
$ErrorActionPreference='Stop'
& (Join-Path $PSScriptRoot '../run_release.ps1') -Action $Action -Targets 'engine:projection' -Destination $Destination -OptionalPlatforms $OptionalPlatforms
