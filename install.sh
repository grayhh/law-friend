#!/usr/bin/env sh
# 법친 (Beopchin) 설치 스크립트 — macOS / Linux
#
# 사용:
#   curl -fsSL https://raw.githubusercontent.com/grayhh/law-friend/main/install.sh | sh
#
# 설치 위치: ~/.beopchin/
#   - ~/.beopchin/beopchin            (실행 바이너리)
#   - ~/.beopchin/data/precedents.json (판례 데이터, 약 254MB)

set -eu

REPO="grayhh/law-friend"
INSTALL_DIR="${BEOPCHIN_HOME:-$HOME/.beopchin}"
DATA_DIR="$INSTALL_DIR/data"
BASE_URL="https://github.com/${REPO}/releases/latest/download"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *)
    echo "✗ 지원하지 않는 CPU 아키텍처: $ARCH" >&2
    exit 1
    ;;
esac
case "$OS" in
  darwin|linux) ;;
  *)
    echo "✗ 지원하지 않는 OS: $OS" >&2
    echo "  Windows는 install.ps1 을 사용하세요." >&2
    exit 1
    ;;
esac

BINARY="beopchin-${OS}-${ARCH}"

mkdir -p "$DATA_DIR"

echo "▶ 법친 바이너리 내려받는 중... ($BINARY)"
curl -fL --progress-bar -o "$INSTALL_DIR/beopchin" "$BASE_URL/$BINARY"
chmod +x "$INSTALL_DIR/beopchin"

if [ "$OS" = "darwin" ]; then
  # Gatekeeper 격리 속성 제거 (없으면 조용히 넘어감)
  xattr -d com.apple.quarantine "$INSTALL_DIR/beopchin" 2>/dev/null || true
fi

if [ -f "$DATA_DIR/precedents.json" ]; then
  echo "▶ 판례 데이터가 이미 있어요. 새로 받으려면 다음 파일을 지우세요:"
  echo "  $DATA_DIR/precedents.json"
else
  echo "▶ 판례 데이터 내려받는 중... (약 254MB, 시간이 좀 걸려요)"
  curl -fL --progress-bar -o "$DATA_DIR/precedents.json" "$BASE_URL/precedents.json"
fi

cat <<EOF

✅ 설치 완료!

실행:
  $INSTALL_DIR/beopchin serve

매번 경로 치기 귀찮으면 PATH에 추가하세요:
  echo 'export PATH="\$HOME/.beopchin:\$PATH"' >> ~/.zshrc   # zsh
  echo 'export PATH="\$HOME/.beopchin:\$PATH"' >> ~/.bashrc  # bash
  # 그 후 'beopchin serve' 로 바로 실행

서버가 뜨면 브라우저에서 http://localhost:8787 을 열어주세요.
EOF
