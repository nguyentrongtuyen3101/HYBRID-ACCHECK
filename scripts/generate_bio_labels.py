"""
Sinh nhãn BIO (token-level) từ dataset Hybrid-ACCheck để fine-tune DeBERTa Token Classification.

QUAN TRỌNG:
  - Ground-truth lấy từ NỘI DUNG LITERAL trong user_story (và AC nếu có).
  - KHÔNG lấy từ permission_matrix (PM có thể đã bị mutate ở case conflict).

Nhãn token:
  O, B-ROLE, I-ROLE, B-ACTION, I-ACTION, B-RESOURCE, I-RESOURCE,
  B-SCOPE, I-SCOPE, B-CONDITION, I-CONDITION, B-EFFECT, I-EFFECT

Chạy:
  python scripts/generate_bio_labels.py \\
    --dataset data/processed/hybrid_accheck_dataset_ecommerce_cms_200.json \\
    --output data/processed/bio_token_labels.json

Sau đó dùng file bio_token_labels.json để fine-tune DeBERTa.
"""
from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Optional


# ---------- Label set ----------
LABELS = [
    "O",
    "B-ROLE",
    "I-ROLE",
    "B-ACTION",
    "I-ACTION",
    "B-RESOURCE",
    "I-RESOURCE",
    "B-SCOPE",
    "I-SCOPE",
    "B-CONDITION",
    "I-CONDITION",
    "B-EFFECT",
    "I-EFFECT",
]

# Connextra-style pattern (tương thích RuleBasedExtractor + câu synthetic)
_CONNEXTRA = re.compile(
    r"(?P<prefix>as\s+an?\s+)"
    r"(?P<role>[\w\s\-]+?)"
    r"(?P<mid>,\s*i\s+(?:want|need|would like)\s+to\s+)"
    r"(?P<action>[\w\-]+)"
    r"\s+"
    r"(?P<resource>[\w\s\-]+?)"
    r"(?P<tail>(?:,?\s*so\s+that|\.|$))",
    re.IGNORECASE,
)

_SIMPLE = re.compile(
    r"(?P<role>[\w\s\-]+?)"
    r"\s+(?:may|can|shall|should be able to|is able to)\s+"
    r"(?P<action>[\w\-]+)"
    r"\s+"
    r"(?P<resource>[\w\s\-]+?)"
    r"(?:,|\.|$)",
    re.IGNORECASE,
)

_SCOPE_WORDS = {
    "assigned": ["assigned"],
    "own": ["own", "my own", "their own"],
    "all": ["all", "any", "every"],
}


def simple_tokenize(text: str) -> list[tuple[str, int, int]]:
    """
    Tokenize đơn giản theo whitespace + giữ punctuation tách riêng.
    Trả về list (token, start_char, end_char).
    """
    tokens = []
    for m in re.finditer(r"\S+", text):
        raw = m.group()
        start = m.start()
        # tách dấu câu cuối token nếu có
        core = raw
        trailing = ""
        while core and core[-1] in ".,;:!?\"')":
            trailing = core[-1] + trailing
            core = core[:-1]
        leading = ""
        while core and core[0] in "\"'(":
            leading += core[0]
            core = core[1:]

        pos = start
        if leading:
            tokens.append((leading, pos, pos + len(leading)))
            pos += len(leading)
        if core:
            tokens.append((core, pos, pos + len(core)))
            pos += len(core)
        if trailing:
            tokens.append((trailing, pos, pos + len(trailing)))
    return tokens


def char_span_to_token_labels(
    tokens: list[tuple[str, int, int]],
    spans: list[tuple[int, int, str]],
) -> list[str]:
    """
    spans: list of (start_char, end_char, label_base) e.g. (10, 22, "ROLE")
    Gán B-/I- theo token overlap.
    """
    labels = ["O"] * len(tokens)
    for start_c, end_c, base in spans:
        first = True
        for i, (_, t_start, t_end) in enumerate(tokens):
            # overlap
            if t_end <= start_c or t_start >= end_c:
                continue
            tag = f"B-{base}" if first else f"I-{base}"
            labels[i] = tag
            first = False
    return labels


def find_span_in_text(text: str, phrase: str) -> Optional[tuple[int, int]]:
    """Tìm vị trí phrase trong text (case-insensitive), trả (start, end) hoặc None."""
    if not phrase:
        return None
    pattern = re.compile(re.escape(phrase), re.IGNORECASE)
    m = pattern.search(text)
    if m:
        return m.start(), m.end()
    return None


def extract_spans_from_user_story(text: str) -> list[tuple[int, int, str]]:
    """
    Trích span Role/Action/Resource/Scope từ literal text.
    Ưu tiên pattern Connextra / Simple.
    """
    spans: list[tuple[int, int, str]] = []
    text_work = text

    m = _CONNEXTRA.search(text_work) or _SIMPLE.search(text_work)
    if m:
        gd = m.groupdict()
        if "role" in gd and gd["role"]:
            # group role nằm trong match
            role_text = gd["role"]
            # tìm offset chính xác trong original
            abs_start = m.start() + m.group(0).lower().find(role_text.lower())
            # fallback: search
            found = find_span_in_text(text, role_text.strip())
            if found:
                spans.append((found[0], found[1], "ROLE"))

        if "action" in gd and gd["action"]:
            found = find_span_in_text(text, gd["action"].strip())
            if found:
                spans.append((found[0], found[1], "ACTION"))

        if "resource" in gd and gd["resource"]:
            # resource có thể bị cắt bởi purpose clause — giữ phần match pattern
            res = gd["resource"].strip().rstrip(".")
            found = find_span_in_text(text, res)
            if found:
                spans.append((found[0], found[1], "RESOURCE"))

    # Scope keywords trong câu
    text_lower = text.lower()
    for scope_label, kws in _SCOPE_WORDS.items():
        for kw in kws:
            idx = text_lower.find(kw)
            if idx >= 0:
                spans.append((idx, idx + len(kw), "SCOPE"))
                break

    # Effect deny keywords
    for kw in ("must not", "cannot", "can not", "should not", "may not", "is not allowed"):
        idx = text_lower.find(kw)
        if idx >= 0:
            spans.append((idx, idx + len(kw), "EFFECT"))
            break

    return spans


def process_item(item: dict) -> dict:
    """Tạo 1 record BIO từ 1 mẫu dataset."""
    text = item["user_story"]
    tokens_info = simple_tokenize(text)
    tokens = [t[0] for t in tokens_info]
    spans = extract_spans_from_user_story(text)
    labels = char_span_to_token_labels(tokens_info, spans)

    # Gộp thêm AC (optional) — mỗi AC 1 record riêng để tăng data
    records = [
        {
            "id": item["id"],
            "source_id": item.get("source_id"),
            "origin": item.get("origin", "unknown"),
            "text": text,
            "tokens": tokens,
            "labels": labels,
            "conflict_label": item.get("conflict_label"),
            "spans_debug": [
                {"start": s, "end": e, "type": t, "text": text[s:e]} for s, e, t in spans
            ],
        }
    ]

    for i, ac in enumerate(item.get("acceptance_criteria") or []):
        if not ac or not ac.strip():
            continue
        ac_tokens_info = simple_tokenize(ac)
        ac_tokens = [t[0] for t in ac_tokens_info]
        ac_spans = extract_spans_from_user_story(ac)
        ac_labels = char_span_to_token_labels(ac_tokens_info, ac_spans)
        records.append(
            {
                "id": f"{item['id']}_AC{i+1}",
                "source_id": item.get("source_id"),
                "origin": item.get("origin", "unknown"),
                "text": ac,
                "tokens": ac_tokens,
                "labels": ac_labels,
                "conflict_label": item.get("conflict_label"),
                "spans_debug": [
                    {"start": s, "end": e, "type": t, "text": ac[s:e]} for s, e, t in ac_spans
                ],
            }
        )

    return records


def main():
    parser = argparse.ArgumentParser(description="Generate BIO labels for DeBERTa fine-tuning")
    parser.add_argument(
        "--dataset",
        default="data/processed/hybrid_accheck_dataset_ecommerce_cms_200.json",
    )
    parser.add_argument(
        "--output",
        default="data/processed/bio_token_labels.json",
    )
    args = parser.parse_args()

    dataset_path = Path(args.dataset)
    if not dataset_path.exists():
        # fallback: try attachments path in sandbox
        alt = Path("/home/workdir/attachments/hybrid_accheck_dataset_ecommerce_cms_200.json")
        if alt.exists():
            dataset_path = alt
        else:
            raise FileNotFoundError(f"Dataset not found: {args.dataset}")

    with open(dataset_path, encoding="utf-8") as f:
        dataset = json.load(f)

    all_records = []
    stats = {"with_role": 0, "with_action": 0, "with_resource": 0, "empty_spans": 0}

    for item in dataset:
        records = process_item(item)
        for rec in records:
            all_records.append(rec)
            labs = set(rec["labels"])
            if any(l.endswith("ROLE") for l in labs):
                stats["with_role"] += 1
            if any(l.endswith("ACTION") for l in labs):
                stats["with_action"] += 1
            if any(l.endswith("RESOURCE") for l in labs):
                stats["with_resource"] += 1
            if labs == {"O"} or not labs:
                stats["empty_spans"] += 1

    out_path = Path(args.output)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(
            {
                "label_list": LABELS,
                "num_records": len(all_records),
                "records": all_records,
            },
            f,
            ensure_ascii=False,
            indent=2,
        )

    print(f"Đã ghi {len(all_records)} records → {out_path}")
    print(f"Stats: {stats}")
    print("\nVí dụ 3 record đầu:")
    for rec in all_records[:3]:
        print(f"  [{rec['id']}] {rec['text'][:60]}...")
        print(f"    tokens: {rec['tokens'][:12]}...")
        print(f"    labels: {rec['labels'][:12]}...")
        print(f"    spans:  {rec['spans_debug']}")


if __name__ == "__main__":
    main()