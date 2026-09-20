@echo off
chcp 65001 >nul
setlocal

echo ======================================================
echo PTSIP 작업 환경 구성을 시작합니다.
echo ======================================================

python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] 파이썬을 찾을 수 없습니다. PATH 설정을 확인하세요.
    pause
    exit /b 1
)

if not exist ".venv" (
    echo [INFO] .venv 가상환경이 없습니다. 새로 생성합니다...
    python -m venv .venv
) else (
    echo [INFO] 기존 .venv 환경을 사용합니다.
)

echo [INFO] 가상환경 활성화 중...
call .venv\Scripts\activate

echo [INFO] pip 최신화 및 PTSIP 개발 의존성 설치 중...
python -m pip install --upgrade pip
if %errorlevel% neq 0 exit /b 1
python -m pip install -e ".[dev]"
if %errorlevel% neq 0 exit /b 1

echo [INFO] PTSIP Git hook 활성화 중...
python -m developer.automation.dev_setup
if %errorlevel% neq 0 (
    echo [ERROR] PTSIP developer hook 설정에 실패했습니다.
    exit /b 1
)

echo.
echo ======================================================
echo PTSIP 개발 환경과 repository-local Git hook 구성이 완료되었습니다.
echo ======================================================
pause
