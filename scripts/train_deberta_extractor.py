"""
Fine-tune DeBERTa-v3-base cho Token Classification (BIO) — bản cải thiện.

Cải thiện:
  - Chỉ dùng User Story (bỏ AC trống)
  - Class weights chống đoán toàn O
  - LR thấp, warmup, gradient clip

Chạy:
  export PYTHONPATH=.
  python scripts/train_deberta_extractor.py \\
    --bio-file data/processed/bio_token_labels.json \\
    --output-dir data/models/deberta-extractor \\
    --epochs 10 --batch-size 8 --lr 1e-5
"""
from __future__ import annotations

import argparse
import json
from collections import Counter
from pathlib import Path

import numpy as np
import torch
from torch import nn
from datasets import Dataset, DatasetDict
from transformers import (
    AutoModelForTokenClassification,
    AutoTokenizer,
    DataCollatorForTokenClassification,
    Trainer,
    TrainingArguments,
    set_seed,
)
import evaluate


def load_bio_records(path: str, us_only: bool = True):
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    records = data["records"]
    filtered = []
    for r in records:
        if not any(l != "O" for l in r["labels"]):
            continue
        if us_only and "_AC" in str(r.get("id", "")):
            continue
        filtered.append(r)
    print(f"Loaded {len(records)} records, kept {len(filtered)} (us_only={us_only})")
    return filtered, data["label_list"]


def build_label_maps(label_list):
    label2id = {l: i for i, l in enumerate(label_list)}
    id2label = {i: l for l, i in label2id.items()}
    return label2id, id2label


def compute_class_weights(records, label2id):
    counter = Counter()
    for r in records:
        for lab in r["labels"]:
            counter[lab] += 1
    n_labels = len(label2id)
    total = sum(counter.values()) or 1
    weights = []
    for i in range(n_labels):
        name = [k for k, v in label2id.items() if v == i][0]
        count = max(counter.get(name, 1), 1)
        w = min(total / (n_labels * count), 20.0)
        weights.append(w)
    mapping = {name: round(weights[label2id[name]], 2) for name in label2id}
    print("Class weights:", mapping)
    return torch.tensor(weights, dtype=torch.float)


def align_labels_with_tokenizer(examples, tokenizer, label2id, max_length):
    tokenized = tokenizer(
        examples["tokens"],
        is_split_into_words=True,
        truncation=True,
        max_length=max_length,
        padding=False,
    )
    all_labels = []
    for i, word_labels in enumerate(examples["labels"]):
        word_ids = tokenized.word_ids(batch_index=i)
        label_ids = []
        previous_word_idx = None
        for word_idx in word_ids:
            if word_idx is None:
                label_ids.append(-100)
            elif word_idx != previous_word_idx:
                lab = word_labels[word_idx] if word_idx < len(word_labels) else "O"
                label_ids.append(label2id.get(lab, label2id["O"]))
            else:
                lab = word_labels[word_idx] if word_idx < len(word_labels) else "O"
                if lab.startswith("B-"):
                    lab = "I-" + lab[2:]
                label_ids.append(label2id.get(lab, label2id["O"]))
            previous_word_idx = word_idx
        all_labels.append(label_ids)
    tokenized["labels"] = all_labels
    return tokenized


class WeightedTrainer(Trainer):
    def __init__(self, class_weights=None, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.class_weights = class_weights

    def compute_loss(self, model, inputs, return_outputs=False, **kwargs):
        labels = inputs.pop("labels")
        outputs = model(**inputs)
        logits = outputs.logits
        weights = self.class_weights
        if weights is not None:
            weights = weights.to(device=logits.device, dtype=logits.dtype)
        loss_fct = nn.CrossEntropyLoss(weight=weights, ignore_index=-100)
        loss = loss_fct(logits.view(-1, model.config.num_labels), labels.view(-1))
        return (loss, outputs) if return_outputs else loss


def compute_metrics_builder(id2label, metric):
    def compute_metrics(eval_pred):
        logits, labels = eval_pred
        predictions = np.argmax(logits, axis=-1)
        true_predictions, true_labels = [], []
        for pred, lab in zip(predictions, labels):
            pred_seq, lab_seq = [], []
            for p, l in zip(pred, lab):
                if l == -100:
                    continue
                pred_seq.append(id2label[int(p)])
                lab_seq.append(id2label[int(l)])
            true_predictions.append(pred_seq)
            true_labels.append(lab_seq)
        results = metric.compute(predictions=true_predictions, references=true_labels)
        return {
            "precision": results.get("overall_precision", 0.0) or 0.0,
            "recall": results.get("overall_recall", 0.0) or 0.0,
            "f1": results.get("overall_f1", 0.0) or 0.0,
            "accuracy": results.get("overall_accuracy", 0.0) or 0.0,
        }
    return compute_metrics


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--bio-file", default="data/processed/bio_token_labels.json")
    parser.add_argument("--output-dir", default="data/models/deberta-extractor")
    parser.add_argument("--model", default="microsoft/deberta-v3-base")
    parser.add_argument("--epochs", type=int, default=10)
    parser.add_argument("--batch-size", type=int, default=8)
    parser.add_argument("--lr", type=float, default=1e-5)
    parser.add_argument("--max-length", type=int, default=128)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--cpu", action="store_true")
    parser.add_argument("--include-ac", action="store_true")
    args = parser.parse_args()

    set_seed(args.seed)
    records, label_list = load_bio_records(args.bio_file, us_only=not args.include_ac)
    label2id, id2label = build_label_maps(label_list)
    class_weights = compute_class_weights(records, label2id)

    ds = Dataset.from_list(
        [{"tokens": r["tokens"], "labels": r["labels"], "id": r["id"]} for r in records]
    )
    split = ds.train_test_split(test_size=0.15, seed=args.seed)
    raw_datasets = DatasetDict({"train": split["train"], "validation": split["test"]})
    print(f"Train: {len(raw_datasets['train'])}, Val: {len(raw_datasets['validation'])}")

    tokenizer = AutoTokenizer.from_pretrained(args.model)
    model = AutoModelForTokenClassification.from_pretrained(
        args.model,
        num_labels=len(label_list),
        id2label=id2label,
        label2id=label2id,
    )

    def tokenize_fn(examples):
        return align_labels_with_tokenizer(examples, tokenizer, label2id, args.max_length)

    tokenized = raw_datasets.map(tokenize_fn, batched=True, remove_columns=["tokens", "id"])
    data_collator = DataCollatorForTokenClassification(tokenizer=tokenizer)
    metric = evaluate.load("seqeval")

    training_args = TrainingArguments(
        output_dir=args.output_dir,
        learning_rate=args.lr,
        per_device_train_batch_size=args.batch_size,
        per_device_eval_batch_size=args.batch_size,
        num_train_epochs=args.epochs,
        weight_decay=0.01,
        eval_strategy="epoch",
        save_strategy="epoch",
        load_best_model_at_end=True,
        metric_for_best_model="f1",
        greater_is_better=True,
        logging_steps=5,
        save_total_limit=2,
        report_to="none",
        use_cpu=args.cpu,
        seed=args.seed,
    )

    trainer = WeightedTrainer(
        class_weights=class_weights,
        model=model,
        args=training_args,
        train_dataset=tokenized["train"],
        eval_dataset=tokenized["validation"],
        processing_class=tokenizer,
        data_collator=data_collator,
        compute_metrics=compute_metrics_builder(id2label, metric),
    )

    print("\n=== Bắt đầu train (weighted CE) ===")
    trainer.train()
    print("\n=== Validation ===")
    metrics = trainer.evaluate()
    print(metrics)

    out = Path(args.output_dir)
    out.mkdir(parents=True, exist_ok=True)
    trainer.save_model(str(out))
    tokenizer.save_pretrained(str(out))
    with open(out / "label_maps.json", "w", encoding="utf-8") as f:
        json.dump(
            {"label2id": label2id, "id2label": {str(k): v for k, v in id2label.items()}},
            f,
            indent=2,
        )
    print(f"\nĐã lưu model → {out}")


if __name__ == "__main__":
    main()