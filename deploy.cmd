@echo off
setlocal
cd /d "%~dp0"

call test.cmd
if errorlevel 1 exit /b 1

if not exist dist mkdir dist

set APP=rsync233
set PKG=./cmd/rsync233

set GOOS=windows
set GOARCH=amd64
go build -trimpath -ldflags="-s -w" -o dist\%APP%_windows_amd64.exe %PKG%
if errorlevel 1 exit /b 1

set GOOS=linux
set GOARCH=amd64
go build -trimpath -ldflags="-s -w" -o dist\%APP%_linux_amd64 %PKG%
if errorlevel 1 exit /b 1

set GOOS=darwin
set GOARCH=arm64
go build -trimpath -ldflags="-s -w" -o dist\%APP%_darwin_arm64 %PKG%
if errorlevel 1 exit /b 1

set GOOS=
set GOARCH=
echo release artifacts written to dist\
