"""국가법령정보센터 Open API 판례 수집 스크립트.

참조: https://www.law.go.kr/LSO/openApiGuide.do
- 검색 API: GET https://www.law.go.kr/DRF/lawSearch.do
- 상세 API: GET https://www.law.go.kr/DRF/lawService.do

주요 파라미터:
- OC: 사용자 인증키 (필수)
- target: 'prec' (판례)
- type: 'JSON' or 'XML'
- display: 한 페이지 결과 수 (max 100)
- page: 페이지 번호 (1-base)
- query: 검색어 (선택; 없으면 전체)
- prncYd: 선고일자 범위 "YYYYMMDD~YYYYMMDD"
- org: 법원종류코드 (e.g. 400201=대법원)

데이터출처:
- "대법원" — 정식 판례, JSON detail 지원 ✅
- "국세법령정보시스템", "지방세법령정보시스템" 등 — detail JSON 미지원, 제외
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
from datetime import date, datetime
from pathlib import Path
from typing import Any

import requests
from dotenv import load_dotenv

SEARCH_URL = "https://www.law.go.kr/DRF/lawSearch.do"
DETAIL_URL = "https://www.law.go.kr/DRF/lawService.do"

DEFAULT_TIMEOUT = 30
POLITE_DELAY = 0.4  # seconds between requests

SOURCE_ALLOW = {"대법원"}  # 데이터출처명 화이트리스트


def get_api_key() -> str:
    load_dotenv(Path(__file__).parent / ".env")
    oc = os.getenv("LAW_GO_KR_OC")
    if not oc:
        print("ERROR: LAW_GO_KR_OC 환경변수가 비어있어요. .env 확인하세요.", file=sys.stderr)
        sys.exit(1)
    return oc


def data_dir() -> Path:
    return Path(__file__).parent.parent / "data"


def search_precedents(
    oc: str,
    *,
    page: int = 1,
    display: int = 100,
    query: str | None = None,
    prnc_yd_range: str | None = None,
    org: str | None = None,
) -> dict[str, Any]:
    params: dict[str, Any] = {
        "OC": oc,
        "target": "prec",
        "type": "JSON",
        "display": display,
        "page": page,
    }
    if query:
        params["query"] = query
    if prnc_yd_range:
        params["prncYd"] = prnc_yd_range
    if org:
        params["org"] = org
    r = requests.get(SEARCH_URL, params=params, timeout=DEFAULT_TIMEOUT)
    r.raise_for_status()
    return r.json()


def get_precedent_detail(oc: str, prec_id: str) -> dict[str, Any]:
    params = {
        "OC": oc,
        "target": "prec",
        "type": "JSON",
        "ID": prec_id,
    }
    r = requests.get(DETAIL_URL, params=params, timeout=DEFAULT_TIMEOUT)
    r.raise_for_status()
    return r.json()


# ---------- Discovery ----------

def cmd_discover(args: argparse.Namespace) -> None:
    oc = get_api_key()
    out_dir = data_dir() / "raw" / "discovery"
    out_dir.mkdir(parents=True, exist_ok=True)

    query = args.query
    print(f"[1/3] 검색 API 호출 (query={query!r}, display={args.display})…")
    search_resp = search_precedents(oc, page=1, display=args.display, query=query)
    (out_dir / "search_sample.json").write_text(
        json.dumps(search_resp, ensure_ascii=False, indent=2), encoding="utf-8"
    )

    ps = search_resp.get("PrecSearch", {})
    items = ps.get("prec", []) or []
    total = ps.get("totalCnt")
    sources: dict[str, int] = {}
    for it in items:
        src = it.get("데이터출처명") or "(unknown)"
        sources[src] = sources.get(src, 0) + 1
    print(f"    totalCnt={total}, 받은 건수={len(items)}, 데이터출처={sources}")

    daebeobwon = [it for it in items if it.get("데이터출처명") == "대법원"]
    if not daebeobwon:
        print("    WARN: '대법원' 출처가 없어요. --query 바꿔서 재시도해 보세요.")
        return

    target = daebeobwon[0]
    prec_id = target.get("판례일련번호")
    print(f"\n[2/3] '대법원' 출처 첫 건 상세 호출: ID={prec_id}")
    time.sleep(POLITE_DELAY)
    detail_resp = get_precedent_detail(oc, prec_id)
    (out_dir / f"detail_sample_{prec_id}.json").write_text(
        json.dumps(detail_resp, ensure_ascii=False, indent=2), encoding="utf-8"
    )

    inner_key = next(iter(detail_resp))
    inner = detail_resp.get(inner_key, {})
    if isinstance(inner, dict):
        print(f"\n=== {inner_key} 필드 ({len(inner)}개) ===")
        for k, v in inner.items():
            preview = str(v)[:80].replace("\n", " ")
            print(f"  - {k}: {preview}{'…' if len(str(v)) > 80 else ''}")

    print("\n[3/3] discovery 완료.")


# ---------- Phase 2A: Index collection ----------

def index_path() -> Path:
    return data_dir() / "raw" / "_index.json"


def load_index() -> dict[str, Any]:
    p = index_path()
    if not p.exists():
        return {"records": {}, "pages_completed": [], "params": None, "last_run": None}
    return json.loads(p.read_text(encoding="utf-8"))


def save_index(idx: dict[str, Any]) -> None:
    p = index_path()
    p.parent.mkdir(parents=True, exist_ok=True)
    tmp = p.with_suffix(".tmp")
    tmp.write_text(json.dumps(idx, ensure_ascii=False, indent=2), encoding="utf-8")
    tmp.replace(p)


def cmd_index(args: argparse.Namespace) -> None:
    oc = get_api_key()

    yfrom = args.year_from
    yto = args.year_to
    prnc_range = f"{yfrom}0101~{yto}1231"
    display = min(args.display, 100)

    print(f"[Phase 2A] 인덱스 수집 — prncYd={prnc_range}, 대법원 출처만 보존")

    idx = load_index()
    # 파라미터가 바뀌면 초기화
    params_now = {"prnc_range": prnc_range, "display": display}
    if idx.get("params") and idx["params"] != params_now:
        print(f"    WARN: 이전 인덱스 파라미터({idx['params']})와 달라요. 새로 시작합니다.")
        idx = {"records": {}, "pages_completed": [], "params": params_now, "last_run": None}
    elif not idx.get("params"):
        idx["params"] = params_now

    records: dict[str, Any] = idx["records"]
    completed_pages = set(idx.get("pages_completed", []))

    # 첫 페이지로 totalCnt 확인
    first = search_precedents(oc, page=1, display=display, prnc_yd_range=prnc_range)
    ps = first.get("PrecSearch", {})
    total = int(ps.get("totalCnt") or 0)
    total_pages = (total + display - 1) // display
    print(f"    totalCnt={total}, 총 페이지={total_pages} (display={display})")

    if args.max_pages:
        total_pages = min(total_pages, args.max_pages)
        print(f"    --max-pages={args.max_pages} 적용 → {total_pages}페이지까지만 수집")

    added = 0
    skipped_source = 0
    for page in range(1, total_pages + 1):
        if page in completed_pages:
            continue
        if page == 1:
            resp = first
        else:
            time.sleep(POLITE_DELAY)
            resp = search_precedents(oc, page=page, display=display, prnc_yd_range=prnc_range)
        items = resp.get("PrecSearch", {}).get("prec", []) or []

        page_added = 0
        for it in items:
            if it.get("데이터출처명") not in SOURCE_ALLOW:
                skipped_source += 1
                continue
            pid = it.get("판례일련번호")
            if not pid or pid in records:
                continue
            records[pid] = {
                "판례일련번호": pid,
                "사건번호": it.get("사건번호"),
                "사건명": it.get("사건명"),
                "법원명": it.get("법원명"),
                "선고일자": it.get("선고일자"),
                "사건종류명": it.get("사건종류명"),
                "사건종류코드": it.get("사건종류코드"),
                "판결유형": it.get("판결유형"),
                "데이터출처명": it.get("데이터출처명"),
            }
            page_added += 1
        added += page_added
        completed_pages.add(page)

        if page % 5 == 0 or page == total_pages:
            idx["pages_completed"] = sorted(completed_pages)
            idx["last_run"] = datetime.now().isoformat()
            save_index(idx)
            print(f"    page {page}/{total_pages} done | 이번 페이지 추가={page_added} | 누적 records={len(records)}")

    idx["pages_completed"] = sorted(completed_pages)
    idx["last_run"] = datetime.now().isoformat()
    save_index(idx)

    print(f"\n[Phase 2A] 완료: records={len(records)}, 추가={added}, 비대법원 skip={skipped_source}")
    print(f"인덱스 파일: {index_path()}")


# ---------- Phase 2B: Detail collection ----------

def prec_raw_dir() -> Path:
    return data_dir() / "raw" / "prec"


def cmd_detail(args: argparse.Namespace) -> None:
    oc = get_api_key()
    out_dir = prec_raw_dir()
    out_dir.mkdir(parents=True, exist_ok=True)

    idx = load_index()
    records: dict[str, Any] = idx.get("records", {})
    if not records:
        print("ERROR: _index.json이 비어 있어요. 먼저 --phase index 를 실행하세요.", file=sys.stderr)
        sys.exit(1)

    ids = list(records.keys())
    print(f"[Phase 2B] 상세 수집 대상 records={len(ids)}")

    todo = [pid for pid in ids if not (out_dir / f"{pid}.json").exists()]
    print(f"    이미 저장됨={len(ids) - len(todo)}, 미수집={len(todo)}")

    if args.max_records:
        todo = todo[: args.max_records]
        print(f"    --max-records={args.max_records} 적용 → {len(todo)}건만 수집")

    fetched = 0
    failed: list[tuple[str, str]] = []
    started = time.time()
    for i, pid in enumerate(todo, start=1):
        try:
            resp = get_precedent_detail(oc, pid)
            inner = resp.get("PrecService")
            if not isinstance(inner, dict) or "판시사항" not in inner:
                failed.append((pid, "invalid response: " + str(resp)[:100]))
                continue
            (out_dir / f"{pid}.json").write_text(
                json.dumps(resp, ensure_ascii=False, indent=2), encoding="utf-8"
            )
            fetched += 1
        except Exception as e:
            failed.append((pid, str(e)[:200]))
        time.sleep(POLITE_DELAY)

        if i % 25 == 0 or i == len(todo):
            elapsed = time.time() - started
            rate = i / elapsed if elapsed > 0 else 0
            remaining = (len(todo) - i) / rate if rate > 0 else 0
            print(
                f"    {i}/{len(todo)} | fetched={fetched} failed={len(failed)} | "
                f"{rate:.1f} req/s | ETA {remaining/60:.1f}분"
            )

    print(f"\n[Phase 2B] 완료: fetched={fetched}, failed={len(failed)}")
    if failed:
        log = out_dir.parent / "_detail_failed.log"
        with log.open("a", encoding="utf-8") as f:
            for pid, msg in failed:
                f.write(f"{datetime.now().isoformat()}\t{pid}\t{msg}\n")
        print(f"실패 로그: {log}")


# ---------- main ----------

def main() -> None:
    p = argparse.ArgumentParser(description="국가법령정보센터 판례 수집")
    sub = p.add_subparsers(dest="mode")

    pd = sub.add_parser("discover", help="응답 구조 확인용 작은 샘플 수집")
    pd.add_argument("--display", type=int, default=20)
    pd.add_argument("--query", type=str, default="손해배상")

    pi = sub.add_parser("index", help="Phase 2A: 검색 인덱스 수집")
    pi.add_argument("--year-from", type=int, default=date.today().year - 10)
    pi.add_argument("--year-to", type=int, default=date.today().year)
    pi.add_argument("--display", type=int, default=100)
    pi.add_argument("--max-pages", type=int, default=0, help="0=제한없음")

    pdet = sub.add_parser("detail", help="Phase 2B: 상세 수집")
    pdet.add_argument("--max-records", type=int, default=0, help="0=제한없음")

    args = p.parse_args()
    if args.mode == "discover":
        cmd_discover(args)
    elif args.mode == "index":
        cmd_index(args)
    elif args.mode == "detail":
        cmd_detail(args)
    else:
        p.print_help()
        sys.exit(2)


if __name__ == "__main__":
    main()
