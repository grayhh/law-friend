"""raw 판례 응답 → Go 앱이 읽을 수 있는 precedents.json 변환.

입력:
  data/raw/_index.json        (검색 메타데이터)
  data/raw/prec/{ID}.json     (상세 PrecService 응답)

출력:
  data/precedents.json        (정제된 Precedent 배열, Go 구조체와 매핑)

처리:
- HTML 태그 제거 (<br/> → \n, 그 외 태그 strip)
- 선고일자 YYYYMMDD → YYYY-MM-DD
- 필드 매핑 (한글 키 → Go json 태그 키)
- detail 없는 record는 제외 (full_text가 비면 RAG에 무의미)
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any


def data_dir() -> Path:
    return Path(__file__).parent.parent / "data"


_HTML_BR_RE = re.compile(r"<\s*br\s*/?\s*>", re.IGNORECASE)
_HTML_TAG_RE = re.compile(r"<[^>]+>")
_MULTI_NL_RE = re.compile(r"\n{3,}")
_MULTI_SPACE_RE = re.compile(r"[ \t]+")


def strip_html(s: Any) -> str:
    if not s:
        return ""
    text = str(s)
    text = _HTML_BR_RE.sub("\n", text)
    text = _HTML_TAG_RE.sub("", text)
    text = text.replace(" ", " ")
    text = _MULTI_NL_RE.sub("\n\n", text)
    text = _MULTI_SPACE_RE.sub(" ", text)
    return text.strip()


def normalize_date(yyyymmdd: Any) -> str:
    if not yyyymmdd:
        return ""
    s = str(yyyymmdd).replace(".", "").replace("-", "")
    if len(s) == 8 and s.isdigit():
        return f"{s[:4]}-{s[4:6]}-{s[6:8]}"
    return str(yyyymmdd)


def to_precedent(detail: dict[str, Any]) -> dict[str, Any] | None:
    ps = detail.get("PrecService")
    if not isinstance(ps, dict):
        return None
    pid = ps.get("판례정보일련번호")
    if not pid:
        return None

    full_text = strip_html(ps.get("판례내용"))
    if not full_text:
        return None

    return {
        "id": f"prec_{pid}",
        "case_number": (ps.get("사건번호") or "").strip(),
        "case_name": (ps.get("사건명") or "").strip(),
        "court": (ps.get("법원명") or "").strip(),
        "decision_date": normalize_date(ps.get("선고일자")),
        "issues": strip_html(ps.get("판시사항")),
        "summary": strip_html(ps.get("판결요지")),
        "full_text": full_text,
        "source_url": f"https://www.law.go.kr/precInfoP.do?precSeq={pid}",
        # 보조 필드 (Go 구조체에는 아직 없지만 RAG·정렬·필터에 유용)
        "case_kind": (ps.get("사건종류명") or "").strip(),
        "case_kind_code": (ps.get("사건종류코드") or "").strip(),
        "judgment_type": (ps.get("판결유형") or "").strip(),
        "references_law": strip_html(ps.get("참조조문")),
        "references_prec": strip_html(ps.get("참조판례")),
    }


def main() -> None:
    p = argparse.ArgumentParser(description="raw 판례 → precedents.json 정제")
    p.add_argument("--output", type=str, default=None, help="출력 경로 (기본: data/precedents.json)")
    p.add_argument("--pretty", action="store_true", help="JSON indent 적용 (디버그용)")
    args = p.parse_args()

    raw_dir = data_dir() / "raw" / "prec"
    out_path = Path(args.output) if args.output else data_dir() / "precedents.json"

    if not raw_dir.exists():
        print(f"ERROR: {raw_dir} 없음. 먼저 collect.py detail을 실행하세요.", file=sys.stderr)
        sys.exit(1)

    files = sorted(raw_dir.glob("*.json"))
    print(f"raw 파일 {len(files)}건 읽는 중…")

    results: list[dict[str, Any]] = []
    skipped = 0
    for f in files:
        try:
            detail = json.loads(f.read_text(encoding="utf-8"))
        except Exception as e:
            print(f"  WARN: {f.name} 파싱 실패 → skip ({e})")
            skipped += 1
            continue
        prec = to_precedent(detail)
        if prec is None:
            skipped += 1
            continue
        results.append(prec)

    # 선고일자 내림차순(최신순) 정렬
    results.sort(key=lambda r: r.get("decision_date") or "", reverse=True)

    out_path.parent.mkdir(parents=True, exist_ok=True)
    if args.pretty:
        out_path.write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
    else:
        out_path.write_text(json.dumps(results, ensure_ascii=False), encoding="utf-8")

    print(f"완료: {len(results)}건 저장 (skipped={skipped})")
    print(f"출력: {out_path}")


if __name__ == "__main__":
    main()
