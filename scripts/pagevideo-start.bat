@echo off
setlocal EnableExtensions EnableDelayedExpansion

rem Local PageVideo launcher. Two modes:
rem   1) Direct:   scripts\pagevideo-start.bat process --input "path\to\video.mp4"
rem   2) Interactive: double-click or run without args -> type args at the pagevideo> prompt.

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

if not "%~1"=="" (
  "%APP%" %*
  set "EXIT_CODE=%ERRORLEVEL%"
  popd
  exit /b !EXIT_CODE!
)

rem --- Interactive mode: no arguments given ---
"%APP%" --help
echo.
echo Type arguments and press Enter to run. Examples:
echo   process --input "D:\Media\lesson.mp4"
echo   process --input "D:\Media\lesson.mp4" --enable-summary
echo   provider check
echo   version
echo Type "exit" or "quit" (or empty line) to close.
echo.

:repl
set "LINE="
set /p "LINE=pagevideo> "
if not defined LINE goto :done
set "TRIMMED=!LINE: =!"
if /i "!TRIMMED!"=="exit" goto :done
if /i "!TRIMMED!"=="quit" goto :done
if /i "!TRIMMED!"=="" goto :repl
call "%APP%" !LINE!
echo.  ^(exit code !ERRORLEVEL!^)
echo.
goto :repl

:done
popd
exit /b 0
