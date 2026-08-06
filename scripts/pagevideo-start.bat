@echo off
setlocal EnableExtensions

rem Local PageVideo launcher. Run from any directory:
rem   scripts\pagevideo-start.bat process --input "path\to\video.mp4"

set "PROJECT_ROOT=%~dp0.."
for %%I in ("%PROJECT_ROOT%") do set "PROJECT_ROOT=%%~fI"
set "APP=%PROJECT_ROOT%\.pagevideo\pagevideo.exe"

pushd "%PROJECT_ROOT%" >nul
if errorlevel 1 (
  echo [PAGEVIDEO] Cannot enter project root: "%PROJECT_ROOT%" 1>&2
  exit /b 1
)

if not exist "%APP%" (
  where go >nul 2>nul
  if errorlevel 1 (
    echo [PAGEVIDEO] Go is not installed and cached binary is missing: "%APP%" 1>&2
    popd
    exit /b 127
  )
  if not exist "%PROJECT_ROOT%\.pagevideo" mkdir "%PROJECT_ROOT%\.pagevideo"
  echo [PAGEVIDEO] Building local CLI...
  go build -o "%APP%" ".\cmd\pagevideo"
  if errorlevel 1 (
    echo [PAGEVIDEO] Build failed. 1>&2
    popd
    exit /b 1
  )
)

if "%~1"=="" (
  "%APP%" --help
  set "EXIT_CODE=%ERRORLEVEL%"
) else (
  "%APP%" %*
  set "EXIT_CODE=%ERRORLEVEL%"
)
popd
exit /b %EXIT_CODE%
