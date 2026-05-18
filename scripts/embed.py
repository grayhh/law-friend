"""Phase B — 판례 임베딩 생성 스크립트 (스켈레톤).

precedents.json → 한국어 sentence-transformer → embeddings.npy + index.json

Go 쪽은 추후 ONNX Runtime + Tokenizer로 동일 모델의 query embedding을 만들고
cosine similarity로 검색.

모델 후보 (작을수록 ONNX export·런타임 비용 적음):
- jhgan/ko-sroberta-multitask    한국어 특화, 768-dim, 약 420MB
- intfloat/multilingual-e5-small 다국어, 384-dim, 약 470MB
- BAAI/bge-m3                    다국어 대형, 1024-dim, 약 2.3GB

사용:
    pip install sentence-transformers numpy tqdm
    python embed.py --model jhgan/ko-sroberta-multitask

출력:
    data/embeddings/
    ├── vectors.npy        float32 (N, D)
    ├── ids.json           ["prec_xxx", ...] 같은 인덱스 (vectors와 같은 순서)
    └── meta.json          {"model": "...", "dim": D, "count": N, "created_at": "..."}
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path


def data_dir() -> Path:
    return Path(__file__).parent.parent / "data"


def build_text(p: dict) -> str:
    """판례 1건을 임베딩 입력 텍스트로 직렬화.

    case_name + issues + summary 를 합치는 게 RAG 신호로 가장 강함.
    full_text는 길고 노이즈가 많아 제외.
    """
    parts = []
    if p.get("case_name"):
        parts.append(f"사건: {p['case_name']}")
    if p.get("case_kind"):
        parts.append(f"종류: {p['case_kind']}")
    if p.get("issues"):
        parts.append(f"판시사항: {p['issues']}")
    if p.get("summary"):
        parts.append(f"판결요지: {p['summary']}")
    return "\n".join(parts)


def main() -> None:
    p = argparse.ArgumentParser(description="판례 임베딩 생성")
    p.add_argument("--model", default="jhgan/ko-sroberta-multitask", help="sentence-transformer 모델 ID")
    p.add_argument("--input", default=str(data_dir() / "precedents.json"))
    p.add_argument("--output", default=str(data_dir() / "embeddings"))
    p.add_argument("--batch-size", type=int, default=32)
    p.add_argument("--dry-run", action="store_true", help="텍스트만 만들고 모델은 안 띄움")
    args = p.parse_args()

    in_path = Path(args.input)
    out_dir = Path(args.output)
    out_dir.mkdir(parents=True, exist_ok=True)

    if not in_path.exists():
        print(f"ERROR: {in_path} 없음", file=sys.stderr)
        sys.exit(1)

    print(f"입력 로드: {in_path}")
    precs = json.loads(in_path.read_text(encoding="utf-8"))
    print(f"판례 {len(precs)}건")

    ids = [p["id"] for p in precs]
    texts = [build_text(p) for p in precs]

    if args.dry_run:
        sample = texts[0][:300] if texts else ""
        print(f"--dry-run: 첫 텍스트 예시\n---\n{sample}\n---")
        return

    # heavy imports lazy
    try:
        import numpy as np
        from sentence_transformers import SentenceTransformer
    except ImportError as e:
        print(
            f"ERROR: {e}. 다음을 설치하세요:\n"
            "  pip install sentence-transformers numpy tqdm",
            file=sys.stderr,
        )
        sys.exit(1)

    print(f"모델 로드: {args.model}")
    model = SentenceTransformer(args.model)

    print(f"임베딩 생성 (batch={args.batch_size})…")
    vectors = model.encode(
        texts,
        batch_size=args.batch_size,
        show_progress_bar=True,
        convert_to_numpy=True,
        normalize_embeddings=True,  # cosine sim을 dot product로 처리 가능
    ).astype("float32")

    vec_path = out_dir / "vectors.npy"
    ids_path = out_dir / "ids.json"
    meta_path = out_dir / "meta.json"
    np.save(vec_path, vectors)
    ids_path.write_text(json.dumps(ids, ensure_ascii=False), encoding="utf-8")
    meta_path.write_text(
        json.dumps(
            {
                "model": args.model,
                "dim": int(vectors.shape[1]),
                "count": int(vectors.shape[0]),
                "normalized": True,
                "input_field_format": "case_name + case_kind + issues + summary",
                "created_at": datetime.now().isoformat(),
            },
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )

    print(f"완료. {vectors.shape} → {vec_path}")
    print(f"size: vectors={vec_path.stat().st_size // (1024*1024)} MB, ids={ids_path.stat().st_size // 1024} KB")


if __name__ == "__main__":
    main()
