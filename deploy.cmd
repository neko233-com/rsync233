@echo off
setlocal
cd /d "%~dp0"

call test.cmd
if errorlevel 1 exit /b 1

if not exist dist mkdir dist

set APP=rsync233
set PKG=./cmd/rsync233
set CGO_ENABLED=0

call :build windows amd64 ".exe" || exit /b 1
call :build windows arm64 ".exe" || exit /b 1
call :build linux amd64 "" || exit /b 1
call :build linux arm64 "" || exit /b 1
call :build darwin amd64 "" || exit /b 1
call :build darwin arm64 "" || exit /b 1

set GOOS=
set GOARCH=
echo release artifacts written to dist\
exit /b 0

:build
set GOOS=%~1
set GOARCH=%~2
set EXT=%~3
echo building %GOOS%/%GOARCH%
go build -trimpath -ldflags="-s -w" -o dist\%APP%_%GOOS%_%GOARCH%%EXT% %PKG%
exit /b %ERRORLEVEL%
