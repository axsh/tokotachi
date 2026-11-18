@echo off
REM Windows wrapper for gitlab-glab-ni.sh (TT_CMD_GLAB on Windows).
setlocal
set "SCRIPT_DIR=%~dp0"
set "BASH="
if exist "%ProgramFiles%\Git\bin\bash.exe" set "BASH=%ProgramFiles%\Git\bin\bash.exe"
if exist "%ProgramFiles%\Git\usr\bin\bash.exe" set "BASH=%ProgramFiles%\Git\usr\bin\bash.exe"
where bash >nul 2>&1 && for /f "delims=" %%i in ('where bash') do if not defined BASH set "BASH=%%i"
if not defined BASH (
  echo bash not found; install Git Bash >&2
  exit /b 1
)
"%BASH%" "%SCRIPT_DIR%gitlab-glab-ni.sh" %*
