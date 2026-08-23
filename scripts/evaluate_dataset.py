"""
Script đánh giá Hybrid-ACCheck trên dataset đã gán nhãn (data/processed/*.json).

Chạy: python scripts/evaluate_dataset.py [--dataset data/processed/hybrid_accheck_dataset_ecommerce_cms_200.json]

In ra:
  - Confusion matrix (dự đoán vs nhãn thật) theo 6 lớp (Normal + 5 loại Conflict)
  - Precision / Recall / F1 cho từng lớp
  - Macro-F1 tổng thể (chỉ số chính dùng để so sánh baseline trong báo cáo)

LƯU Ý: Script này gọi model SBERT thật (qua HybridACCheckPipeline mặc định).
Cần đã `pip install -r requirements.txt` và có mạng để tải model lần đầu.
"""
import argparse
import asyncio
import json
import sys
from collections import defaultdict
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from app.domain.enums.conflict import EffectType
from app.domain.schemas.access_control import PermissionMatrixRowSchema, PipelineInput
from app.services.pipeline import HybridACCheckPipeline

ALL_LABELS = [
    "Normal",
    "Missing Permission",
    "Action Conflict",
    "Effect Conflict",
    "Scope Conflict",
    "Condition Conflict",
]


def load_dataset(path: str) -> list[dict]:
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def to_payload(item: dict) -> PipelineInput:
    pm_rows = [
        PermissionMatrixRowSchema(
            role=r["role"],
            action=r["action"],
            resource=r["resource"],
            effect=EffectType(r["effect"]),
            condition=r.get("condition"),
            scope=r.get("scope"),
        )
        for r in item["permission_matrix"]
    ]
    return PipelineInput(
        user_story=item["user_story"],
        acceptance_criteria=item.get("acceptance_criteria", []),
        permission_matrix=pm_rows,
    )


def compute_prf1(confusion: dict, label: str) -> tuple[float, float, float]:
    tp = confusion[label][label]
    fp = sum(confusion[other][label] for other in ALL_LABELS if other != label)
    fn = sum(confusion[label][other] for other in ALL_LABELS if other != label)

    precision = tp / (tp + fp) if (tp + fp) > 0 else 0.0
    recall = tp / (tp + fn) if (tp + fn) > 0 else 0.0
    f1 = (2 * precision * recall / (precision + recall)) if (precision + recall) > 0 else 0.0
    return precision, recall, f1


async def evaluate(dataset_path: str) -> None:
    dataset = load_dataset(dataset_path)
    pipeline = HybridACCheckPipeline()

    # confusion[expected][predicted] = count
    confusion: dict = defaultdict(lambda: defaultdict(int))
    errors = []

    print(f"Đang đánh giá {len(dataset)} mẫu từ {dataset_path} ...\n")

    for item in dataset:
        payload = to_payload(item)
        result = await pipeline.run(payload)
        predicted = result.conflicts[0].conflict_type.value if result.conflicts else "Normal"
        expected = item["conflict_label"]

        confusion[expected][predicted] += 1
        if predicted != expected:
            errors.append((item["id"], expected, predicted, item["user_story"][:70]))

    # ---- In confusion matrix ----
    print("=== CONFUSION MATRIX (hàng = nhãn thật, cột = dự đoán) ===")
    header = " " * 22 + "".join(f"{lbl[:10]:>12}" for lbl in ALL_LABELS)
    print(header)
    for expected in ALL_LABELS:
        row = f"{expected:22}" + "".join(f"{confusion[expected][pred]:>12}" for pred in ALL_LABELS)
        print(row)

    # ---- In Precision/Recall/F1 từng lớp ----
    print("\n=== PRECISION / RECALL / F1 theo từng lớp ===")
    print(f"{'Label':22}{'Precision':>12}{'Recall':>12}{'F1':>12}{'Support':>10}")
    f1_scores = []
    for label in ALL_LABELS:
        support = sum(confusion[label].values())
        if support == 0:
            continue
        p, r, f1 = compute_prf1(confusion, label)
        f1_scores.append(f1)
        print(f"{label:22}{p:>12.3f}{r:>12.3f}{f1:>12.3f}{support:>10}")

    macro_f1 = sum(f1_scores) / len(f1_scores) if f1_scores else 0.0
    total = len(dataset)
    correct = sum(confusion[lbl][lbl] for lbl in ALL_LABELS)
    print(f"\nAccuracy tổng thể: {correct}/{total} = {correct/total*100:.1f}%")
    print(f"Macro-F1: {macro_f1:.3f}  <-- chỉ số chính dùng để so sánh baseline trong báo cáo")

    # ---- In các case sai để debug ----
    if errors:
        print(f"\n=== {len(errors)} CASE DỰ ĐOÁN SAI (để rà soát) ===")
        for id_, exp, pred, text in errors:
            print(f"[{id_}] expected={exp:20s} predicted={pred:20s} | {text}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Đánh giá Hybrid-ACCheck trên dataset đã gán nhãn")
    parser.add_argument(
        "--dataset",
        default="data/processed/hybrid_accheck_dataset_ecommerce_cms_200.json",
        help="Đường dẫn tới file dataset JSON (mặc định: data/processed/hybrid_accheck_dataset_ecommerce_cms_200.json)",
    )
    args = parser.parse_args()
    asyncio.run(evaluate(args.dataset))