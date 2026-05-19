# 법친 (Beopchin)

> 친구처럼 편한 한국어 AI 법률 친구. 대법원 판례를 검색해서 근거와 함께 답해줍니다.

![법친 화면](sample.png)

법친은 **내 컴퓨터에서 직접 돌아가는** 프로그램이에요. 질문 내용이 외부 서버로 따로 저장되지 않고, 평소 쓰던 **Claude** 또는 **Codex** CLI를 그대로 사용합니다.

> ⚠️ 법친이 주는 답은 **일반적인 법률 정보**예요. 변호사의 법률 자문이 아니니, 실제 사건은 꼭 변호사와 상담하세요.

---

## 필요한 환경

1. **Claude CLI** 또는 **Codex CLI** 중 하나 — 실제로 답을 생성하는 AI 엔진이에요. 둘 다 있어도 좋고, 화면에서 골라 쓸 수 있어요.
   - Claude CLI: <https://docs.claude.com/en/docs/claude-code>
   - Codex CLI: <https://github.com/openai/codex>
2. 디스크 약 **300MB** 여유 (판례 데이터 254MB + 바이너리)

법친은 사용자가 이미 로그인해 둔 CLI를 그대로 실행해서 답을 받아와요. 별도의 API 키를 법친에 넣을 필요는 없어요. **Go 같은 개발 도구는 안 깔아도 됩니다** — 미리 빌드된 바이너리를 받아서 바로 실행하면 끝이에요.

---

## 설치하기

### 🍎 macOS / 🐧 Linux

터미널에 한 줄 붙여넣기:

```bash
curl -fsSL https://raw.githubusercontent.com/grayhh/law-friend/main/install.sh | sh
```

설치가 끝나면 안내된 명령으로 실행하세요:

```bash
~/.beopchin/beopchin serve
```

### 🪟 Windows

PowerShell에 한 줄 붙여넣기:

```powershell
irm https://raw.githubusercontent.com/grayhh/law-friend/main/install.ps1 | iex
```

설치가 끝나면:

```powershell
& "$HOME\.beopchin\beopchin.exe" serve
```

### 🛠 수동 설치 (스크립트가 싫을 때)

1. [Releases 페이지](https://github.com/grayhh/law-friend/releases/latest)에서 본인 OS용 바이너리와 `precedents.json` 을 받아요.
   - macOS Apple Silicon: `beopchin-darwin-arm64`
   - macOS Intel: `beopchin-darwin-amd64`
   - Windows: `beopchin-windows-amd64.exe`
   - Linux x86_64: `beopchin-linux-amd64`
   - Linux ARM64: `beopchin-linux-arm64`
2. 바이너리 옆에 `data/precedents.json` 으로 판례 파일을 놓아요. 예:
   ```
   beopchin/
   ├── beopchin              (또는 beopchin.exe)
   └── data/
       └── precedents.json
   ```
3. 바이너리를 실행하면 자동으로 옆 `data/` 폴더의 판례를 불러와요.
   - macOS는 첫 실행 시 "확인되지 않은 개발자" 경고가 떠요. 터미널에서 한 번만:
     ```bash
     xattr -d com.apple.quarantine ./beopchin
     ```

---

## 사용하기

1. 위 단계로 서버를 띄우면 터미널에 이렇게 뜹니다.
   ```
   법친이 준비됐어요! → http://localhost:8787
   ```
2. 브라우저에서 **<http://localhost:8787>** 을 열면 채팅 화면이 나와요.
3. 오른쪽 위에서 **엔진(Claude / Codex)** 을 골라요.
4. 아래 입력칸에 평소 말투로 질문을 던지면 됩니다.

   > 예) "전세 보증금을 집주인이 안 돌려주는데 어떡하지?"

5. 답변 아래에 **📚 참고 판례** 가 같이 표시돼요. 사건번호를 누르면 원문을 볼 수 있어요.

서버를 끄려면 터미널에서 `Ctrl + C` 를 누르세요.

---

## 대화 기록은 어디에 저장돼요?

전부 내 컴퓨터 안에만 있어요. 외부로 나가지 않습니다.

- **macOS / Linux**: `~/.beopchin/sessions.json`
- **Windows**: `%USERPROFILE%\.beopchin\sessions.json`

대화를 지우고 싶다면 이 파일을 지우면 돼요.

---

## 업데이트하기

설치 스크립트를 다시 실행하면 최신 바이너리로 덮어써져요. 판례 데이터는 이미 받아둔 게 있으면 건너뛰니까 금방 끝납니다.

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/grayhh/law-friend/main/install.sh | sh
```

```powershell
# Windows
irm https://raw.githubusercontent.com/grayhh/law-friend/main/install.ps1 | iex
```

---

## 제거하기

설치 폴더만 지우면 끝이에요.

```bash
# macOS / Linux
rm -rf ~/.beopchin

# Windows (PowerShell)
Remove-Item -Recurse -Force "$HOME\.beopchin"
```

---

## 문제가 생겼을 때

| 증상 | 해결 |
|------|------|
| macOS: "확인되지 않은 개발자" 경고 | 터미널에서 `xattr -d com.apple.quarantine ~/.beopchin/beopchin` 한 번 실행 (스크립트 설치는 자동 처리). |
| `claude/codex CLI를 찾을 수 없어요` | 해당 CLI가 설치돼 있고 `claude --version` / `codex --version` 이 동작하는지 확인하세요. |
| 8787 포트가 이미 쓰여요 | 다른 프로그램이 사용 중이에요. 그 프로그램을 끄거나, 법친을 다시 실행하세요. |
| 검색 결과가 이상해요 | `~/.beopchin/data/precedents.json` 파일이 있는지 확인하세요. 없다면 설치 스크립트를 다시 실행하세요. |

---

## 개발자용: 소스에서 빌드

법친에 기여하거나 코드를 직접 고치고 싶다면:

```bash
# Go 1.25+ 필요
git clone https://github.com/grayhh/law-friend.git
cd law-friend

# 판례 데이터 받기
mkdir -p data
curl -L -o data/precedents.json \
  https://github.com/grayhh/law-friend/releases/latest/download/precedents.json

# 빌드 + 실행
go build -o beopchin .
./beopchin serve
```

---

## 데이터 출처

- 국가법령정보센터 Open API (<https://www.law.go.kr>)
- 대법원 판례 약 1만 4천여 건 (2016~2025년 선고분)

---

## 라이선스

이 프로젝트는 개인적·학습 목적으로 자유롭게 쓸 수 있어요. 상업적 이용 전에는 데이터 출처(국가법령정보센터)의 이용 약관을 꼭 확인하세요.
