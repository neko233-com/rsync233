@echo off
setlocal
cd /d "%~dp0"

gofmt -w cmd internal
if errorlevel 1 exit /b 1

go mod tidy
if errorlevel 1 exit /b 1

go vet ./...
if errorlevel 1 exit /b 1

where gcc >nul 2>nul
if errorlevel 1 (
  echo gcc not found; running tests without -race
  set CGO_ENABLED=0
  go test -count=1 ./...
) else (
  set CGO_ENABLED=1
  go test -race -count=1 ./...
)
if errorlevel 1 exit /b 1

set CGO_ENABLED=0
go build -trimpath -o "%TEMP%\rsync233-test.exe" ./cmd/rsync233
if errorlevel 1 exit /b 1
del "%TEMP%\rsync233-test.exe" >nul 2>nul

call check-platform-matrix.cmd
if errorlevel 1 exit /b 1

call verify-actions.cmd
if errorlevel 1 exit /b 1

echo rsync233 test ok
