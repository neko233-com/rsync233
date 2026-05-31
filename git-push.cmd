@echo off
setlocal
cd /d "%~dp0"

call test.cmd
if errorlevel 1 exit /b 1

git status --short
git add .
git diff --cached --quiet
if errorlevel 1 (
  git commit -m "Initial rsync233 implementation"
  if errorlevel 1 exit /b 1
) else (
  echo no staged changes to commit
)

git push -u origin main
