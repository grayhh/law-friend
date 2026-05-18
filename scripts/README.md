# 법친 빌드 파이프라인 (시스템 A)

판례 데이터를 국가법령정보센터 Open API에서 수집·정제하여 법친 Go 앱이 사용할 형태로 저장합니다.

## 사전 준비

1. **API 키 발급** — https://www.law.go.kr/LSO/openApiGuide.do
2. `.env` 파일 생성:
   ```bash
   cp .env.example .env
   # .env 열어서 LAW_GO_KR_OC=<발급받은_OC> 설정
   ```
3. Python 의존성 설치:
   ```bash
   python3 -m venv .venv
   source .venv/bin/activate
   pip install -r requirements.txt
   ```

## 실행

### Phase 1 — Discovery (응답 구조 확인)

```bash
python collect.py --discover
```

- 작은 샘플(10건)을 수집해 응답 필드 구조를 출력
- `../data/raw/discovery/`에 원본 JSON 저장

### Phase 2 — 본 수집 (추후)

```bash
python collect.py --collect --year-from 2016 --year-to 2025
```

(아직 미구현. Phase 1 결과 보고 진행)

## 데이터 흐름

```
국가법령정보센터 API
    ↓ collect.py
data/raw/{date}/*.json  (원본)
    ↓ normalize.py  (미구현)
data/precedents.json  (Go가 읽는 정제된 형식)
```
