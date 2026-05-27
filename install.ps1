# 로메이트 (Lawmate) 설치 스크립트 — Windows (PowerShell)
#
# 사용:
#   irm https://raw.githubusercontent.com/grayhh/law-friend/main/install.ps1 | iex
#
# 설치 위치: %USERPROFILE%\.lawmate\
#   - %USERPROFILE%\.lawmate\lawmate.exe          (실행 바이너리)
#   - %USERPROFILE%\.lawmate\data\precedents.json  (판례 데이터, 약 254MB)

$ErrorActionPreference = 'Stop'

$Repo       = 'grayhh/law-friend'
$InstallDir = if ($env:LAWMATE_HOME) { $env:LAWMATE_HOME } else { Join-Path $HOME '.lawmate' }
$DataDir    = Join-Path $InstallDir 'data'
$BaseUrl    = "https://github.com/$Repo/releases/latest/download"

# CPU 아키텍처 감지
$archRaw = $env:PROCESSOR_ARCHITECTURE
switch -Regex ($archRaw) {
    'AMD64|x86_64' { $arch = 'amd64' }
    'ARM64'        { $arch = 'arm64' }
    default {
        Write-Error "지원하지 않는 CPU 아키텍처: $archRaw"
        exit 1
    }
}

# 현재 릴리스에 windows-arm64가 없으므로 ARM64는 amd64로 폴백(에뮬레이션)
if ($arch -eq 'arm64') {
    Write-Host "ⓘ Windows ARM64는 amd64 바이너리로 실행됩니다 (에뮬레이션)." -ForegroundColor Yellow
    $arch = 'amd64'
}

$binary = "lawmate-windows-$arch.exe"

New-Item -ItemType Directory -Force -Path $DataDir | Out-Null

$exePath = Join-Path $InstallDir 'lawmate.exe'
Write-Host "▶ 로메이트 바이너리 내려받는 중... ($binary)"
Invoke-WebRequest -Uri "$BaseUrl/$binary" -OutFile $exePath

$dataPath = Join-Path $DataDir 'precedents.json'
if (Test-Path $dataPath) {
    Write-Host "▶ 판례 데이터가 이미 있어요. 새로 받으려면 다음 파일을 지우세요:"
    Write-Host "  $dataPath"
} else {
    Write-Host "▶ 판례 데이터 내려받는 중... (약 254MB, 시간이 좀 걸려요)"
    Invoke-WebRequest -Uri "$BaseUrl/precedents.json" -OutFile $dataPath
}

Write-Host ""
Write-Host "✅ 설치 완료!" -ForegroundColor Green
Write-Host ""
Write-Host "실행:"
Write-Host "  $exePath serve"
Write-Host ""
Write-Host "매번 경로 치기 귀찮으면 PATH에 추가하세요:"
Write-Host "  [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$InstallDir', 'User')"
Write-Host "  # PowerShell 새로 연 뒤 'lawmate serve' 로 실행"
Write-Host ""
Write-Host "서버가 뜨면 브라우저에서 http://localhost:8787 을 열어주세요."
