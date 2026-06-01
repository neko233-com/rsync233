@echo off
setlocal
cd /d "%~dp0"

call test.cmd
if errorlevel 1 exit /b 1

set "VERSION=%~1"
if "%VERSION%"=="" (
  if exist version.txt (
    set /p VERSION=<version.txt
  ) else (
    set "VERSION=dev"
  )
)
if "%VERSION:~0,1%"=="v" (
  set "VERSION_FLAG=%VERSION%"
) else (
  set "VERSION_FLAG=v%VERSION%"
)

if exist dist rmdir /s /q dist
mkdir dist

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
go build -trimpath -ldflags="-s -w -X main.version=%VERSION_FLAG%" -o dist\%APP%-%GOOS%-%GOARCH%%EXT% %PKG%
exit /b %ERRORLEVEL%
