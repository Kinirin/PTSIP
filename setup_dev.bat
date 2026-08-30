@echo off
chcp 65001 >nul
setlocal

echo ======================================================
echo PTSIP 작업 환경 구성을 시작합니다.
echo ======================================================

:: 1. 파이썬 설치 확인
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] 파이썬을 찾을 수 없습니다. PATH 설정을 확인하세요.
    pause
    exit /b
)

:: 2. 가상환경(.venv) 존재 여부 확인 및 생성
if not exist ".venv" (
    echo [INFO] .venv 가상환경이 없습니다. 새로 생성합니다...
    python -m venv .venv
) else (
    echo [INFO] 기존 .venv 환경을 사용합니다.
)

:: 3. 가상환경 활성화 및 의존성 설치
echo [INFO] 가상환경 활성화 중...
call .venv\Scripts\activate

echo [INFO] pip 최신화 및 패키지 설치 중 (requirements.txt)...
python -m pip install --upgrade pip

if exist "requirements.txt" (
    :: 기존 requirements.txt를 기반으로 설치
    pip install -r requirements.txt
) else (
    echo [WARN] requirements.txt가 없습니다. 기본 PyQt6를 설치합니다.
    pip install PyQt6
)

:: 4. 설치 결과 검증
echo.
echo [검증] 설치된 주요 패키지 버전:
python -c "import PyQt6.QtCore; from PyQt6.QtCore import PYQT_VERSION_STR; print(f'- PyQt6 버전: {PYQT_VERSION_STR}')"
python -c "import fastapi; print(f'- FastAPI 버전: {fastapi.__version__}')"

echo.
echo ======================================================
echo 이제 ".venv" 환경에서 안전하게 개발하실 수 있습니다.
echo ======================================================
pause