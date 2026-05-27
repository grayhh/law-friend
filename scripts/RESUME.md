# 판례 수집 이어받기

다른 세션·다른 시점에 판례 detail 수집을 이어서 받기 위한 가이드.

## 현재 상태 (스냅샷)

| 항목 | 값 |
|---|---|
| 대상 인덱스 | `data/raw/_index.json` — 14,470건 (대법원 출처, 2016~2025) |
| 1차 수집 목표 | 3,000건 |
| 1차 완료 후 남은 | 약 11,470건 |

진행률은 언제든 `data/raw/prec/` 폴더 안의 파일 개수로 확인 가능:

```bash
ls /Users/chan/chdev/ch/lawmate/data/raw/prec/ | wc -l
```

## 사전 준비 (한 번만)

1. **API 키** — `scripts/.env`에 `LAW_GO_KR_OC=<발급키>`가 있어야 함.
2. **Python venv** — `scripts/.venv/`에 이미 만들어져 있음. 사라졌다면:
   ```bash
   cd /Users/chan/chdev/ch/lawmate/scripts
   python3 -m venv .venv
   source .venv/bin/activate
   pip install -r requirements.txt
   ```

## 이어받기 (가장 흔한 시나리오)

남은 전부를 받고 끝나면 자동으로 `data/precedents.json`까지 갱신:

```bash
cd /Users/chan/chdev/ch/lawmate/scripts
source .venv/bin/activate
python collect.py detail && python normalize.py
```

- `collect.py detail`은 `data/raw/prec/`에 없는 ID만 받음 → 이미 저장된 건 자동 skip
- 약 9,000건 남은 경우 0.4초/요청 기준 **약 1~1.5시간** 예상
- 중간에 Ctrl+C로 끊어도 안전. 다시 같은 명령으로 이어받기 가능

### 일부만 받고 끊고 싶을 때

`--max-records N`은 *이번 실행에서 새로 받을 건수*를 제한:

```bash
python collect.py detail --max-records 1000 && python normalize.py
```

### 백그라운드로 돌리고 싶을 때

```bash
cd /Users/chan/chdev/ch/lawmate/scripts
source .venv/bin/activate
nohup bash -c 'python -u collect.py detail && python normalize.py' \
  > /tmp/lawmate-collect.log 2>&1 &
echo $! > /tmp/lawmate-collect.pid
```

진행 보기:
```bash
tail -f /tmp/lawmate-collect.log
ls /Users/chan/chdev/ch/lawmate/data/raw/prec/ | wc -l
```

중단:
```bash
kill $(cat /tmp/lawmate-collect.pid) 2>/dev/null
# 또는
pkill -f "collect.py detail"
```

## 끝난 뒤

1. `data/precedents.json`이 새 데이터로 갱신됨 (normalize 자동 실행됨)
2. Go 서버는 외부 파일 우선 로드라 **재빌드 없이도** 재시작만 하면 새 데이터 사용:
   ```bash
   cd /Users/chan/chdev/ch/lawmate
   ./lawmate serve
   ```
3. 검색이 잘 되는지 빠른 확인:
   ```bash
   curl -s "http://localhost:8787/api/search?q=전세+보증금&k=5" | python3 -m json.tool
   ```

## 트러블슈팅

- **`ERROR: LAW_GO_KR_OC 환경변수가 비어있어요.`**
  → `scripts/.env`에 키 설정. 발급: https://www.law.go.kr/LSO/openApiGuide.do
- **`requests.exceptions.HTTPError: 429`** (rate limit)
  → POLITE_DELAY 늘리기 (`scripts/collect.py` 상단의 `POLITE_DELAY = 0.4` → 1.0 등)
- **개별 ID에서 `invalid response` 실패가 누적**
  → `data/raw/_detail_failed.log` 확인. 일시 오류라면 그 ID를 지우고 재실행하면 다시 시도됨
- **`_index.json`이 비어 있다고 나옴**
  → 인덱스부터 다시: `python collect.py index --year-from 2016 --year-to 2025`

## 인덱스 자체를 새로고치고 싶을 때

새 판례가 추가됐거나 다른 기간을 추가하고 싶을 때:

```bash
cd /Users/chan/chdev/ch/lawmate/scripts
source .venv/bin/activate
python collect.py index --year-from 2016 --year-to 2025
```

기존 `_index.json` 파라미터와 다르면 자동으로 새로 시작함.
