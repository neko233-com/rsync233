@echo off
setlocal
cd /d "%~dp0"

node --version
if errorlevel 1 exit /b 1

call npm ci
if errorlevel 1 exit /b 1

call npm run verify:workflows
if errorlevel 1 exit /b 1

echo github actions verification ok
