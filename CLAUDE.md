# 법친 (Beopchin)

친구처럼 편한 한국어 AI 법률 정보 도우미. 사용자가 자기 로컬에서 실행하는 프로그램(SaaS 아님).

## 아키텍처 한눈에

```
사용자 질문 (브라우저 채팅 UI)
  → Go 서버 (localhost:8787, net/http)
    → BM25 검색 (internal/search/) — 현재 단계
    → 프롬프트 구성 (internal/prompt/, 시스템 프롬프트 + 판례 컨텍스트)
    → 사용자 PC의 Claude CLI 또는 Codex CLI를 os/exec로 호출
  → 답변 + 인용 판례 표시
```

- **언어**: Go (사용자 앱), Python (데이터 파이프라인)
- **벡터 DB / 임베딩**: 아직 없음. BM25 lexical search 사용. semantic은 Phase B
- **LLM 호출**: 사용자가 UI에서 Claude/Codex 선택 → Go가 해당 CLI 실행
- **세션 저장**: `~/.beopchin/sessions.json` (서버 측 atomic write)

## 디렉터리

```
beopchin/
├── main.go                      엔트리포인트, serve 명령
├── beopchin                     빌드된 바이너리
├── internal/
│   ├── precedents/              Precedent 타입 + Store 인터페이스 + DummyStore (BM25 백엔드)
│   ├── search/                  BM25 lexical search (한글 토큰화 포함)
│   ├── prompt/                  시스템 프롬프트 + RAG 컨텍스트 포맷
│   ├── llm/                     Claude/Codex CLI 어댑터 + 자동 감지
│   ├── sessions/                대화 세션 영속화
│   ├── server/                  HTTP 핸들러
│   └── data/                    embed용 더미 데이터 (1건)
├── web/                         정적 프론트엔드 (embed로 번들)
├── data/
│   ├── precedents.json          정제된 판례 (Go가 읽음, normalize.py가 생성)
│   ├── raw/                     수집 원본 (gitignore, 개발 머신에만)
│   └── embeddings/              Phase B 임베딩 산출물 (gitignore)
└── scripts/                     Python 데이터 파이프라인
    ├── collect.py               국가법령정보센터 API → raw JSON
    ├── normalize.py             raw → data/precedents.json (HTML 제거, 매핑)
    ├── embed.py                 (Phase B) 임베딩 생성 스켈레톤
    └── RESUME.md                판례 수집 이어받기 가이드
```

## 자주 쓰는 명령

```bash
# 빌드 + 실행
go build -o beopchin . && ./beopchin serve

# BM25 검색 미리보기 (LLM 없이)
curl -s "http://localhost:8787/api/search?q=전세+보증금&k=5" | python3 -m json.tool

# 단위 테스트
go test ./internal/search/ -v

# 판례 수집 이어받기
cd scripts && source .venv/bin/activate
python collect.py detail && python normalize.py
```

## 데이터 출처

- 국가법령정보센터 Open API (law.go.kr/DRF/)
- 인증키: `scripts/.env`의 `LAW_GO_KR_OC`
- 데이터출처명 == "대법원" 인 판례만 사용 (다른 출처는 JSON detail 미지원)

## 디자인 톤

- 친근/캐주얼한 한국어 ("~해줘", "~지" 같은 친구 톤)
- 답변 끝에 **반드시** 면책 고지 부착 (법률 자문 아님)
- 인용 판례는 **반드시** RAG로 검색된 실제 판례만 — 모델이 사건번호를 지어내는 것은 시스템 레벨 금지
- 브랜드 컬러: 녹색 `#4a7c59`
