@echo off
echo === suanming-agent ===

REM load .env
for /f "tokens=1,2 delims==" %%a in (.env) do (
    echo %%a | findstr /b "#" >nul || set "%%a=%%b"
)

if "%LLM_API_KEY%"=="" (
    echo ERROR: LLM_API_KEY not set in .env
    pause
    exit /b 1
)

echo [1/3] knowledge (:3100)
start "knowledge" cmd /c "cd knowledge && npx next dev -p 3100 --turbo"
timeout /t 4 /nobreak >nul

echo [2/3] backend (:8080)
start "backend" cmd /c "go run ./backend/cmd/server/"
timeout /t 2 /nobreak >nul

echo [3/3] frontend (:5173)
start "frontend" cmd /c "cd web && npm run dev"

echo.
echo frontend : http://localhost:5173
echo backend  : http://localhost:8080/api/health
echo knowledge: http://localhost:3100
echo.
echo Press any key to stop all services...
pause >nul
taskkill /f /fi "WINDOWTITLE eq knowledge*" >nul 2>&1
taskkill /f /fi "WINDOWTITLE eq backend*" >nul 2>&1
taskkill /f /fi "WINDOWTITLE eq frontend*" >nul 2>&1
