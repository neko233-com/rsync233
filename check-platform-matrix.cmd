@echo off
setlocal
cd /d "%~dp0"

set PKG=./cmd/rsync233
set OUTDIR=%TEMP%\rsync233-platform-check

if exist "%OUTDIR%" rmdir /s /q "%OUTDIR%"
mkdir "%OUTDIR%"

call :build windows amd64 ".exe" || exit /b 1
call :build windows arm64 ".exe" || exit /b 1
call :build linux amd64 "" || exit /b 1
call :build linux arm64 "" || exit /b 1
call :build darwin amd64 "" || exit /b 1
call :build darwin arm64 "" || exit /b 1

set GOOS=
set GOARCH=
rmdir /s /q "%OUTDIR%"
echo platform matrix ok: windows/linux/darwin x amd64/arm64
exit /b 0

:build
set GOOS=%~1
set GOARCH=%~2
set EXT=%~3
echo building %GOOS%/%GOARCH%
go build -trimpath -o "%OUTDIR%\rsync233_%GOOS%_%GOARCH%%EXT%" %PKG%
exit /b %ERRORLEVEL%
