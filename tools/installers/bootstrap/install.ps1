# Bootstrap installer for tt
# Usage: iwr -useb <url>/install.ps1 | iex

$ToolName = "tt"
$Version = "v0.6.0"
$BaseURL = "https://github.com/axsh/tokotachi/releases/download/v0.6.0"

$ArchiveURL = "${BaseURL}/${ToolName}_windows_amd64.zip"
$InstallDir = "$env:LOCALAPPDATA\${ToolName}\bin"

Write-Host "Installing ${ToolName} ${Version} for windows_amd64..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$TempFile = [System.IO.Path]::GetTempFileName() + ".zip"
Invoke-WebRequest -Uri $ArchiveURL -OutFile $TempFile
Expand-Archive -Path $TempFile -DestinationPath $InstallDir -Force
Remove-Item $TempFile
Write-Host "Installed to ${InstallDir}\${ToolName}.exe"
