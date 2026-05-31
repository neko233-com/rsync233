@echo off
setlocal
cd /d "%~dp0"

gofmt -w cmd internal
if errorlevel 1 exit /b 1

go mod tidy
if errorlevel 1 exit /b 1

go test ./...
if errorlevel 1 exit /b 1

go build -o "%TEMP%\rsync233-test.exe" ./cmd/rsync233
if errorlevel 1 exit /b 1
del "%TEMP%\rsync233-test.exe" >nul 2>nul

call check-platform-matrix.cmd
if errorlevel 1 exit /b 1

echo rsync233 test ok
