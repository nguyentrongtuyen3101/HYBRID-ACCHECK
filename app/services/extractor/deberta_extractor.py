"""
Requirement Extractor dùng DeBERTa Token Classification (fine-tuned).
Load checkpoint từ data/models/deberta-extractor/
Theo đúng thiết kế đề tài: DeBERTa Token Classification.
"""
from __future__ import annotations

import json
from pathlib import Path

import torch
from loguru import logger
from transformers import AutoModelForTokenClassification, AutoTokenizer

from app.domain.enums.conflict import ArtifactSource, EffectType
from app.domain.entities.access_control import ExtractedEntity
from app.services.extractor.base import BaseExtractor

DEFAULT_MODEL_DIR = "data/models/deberta-extractor"


class DeBERTaExtractor(BaseExtractor):
    def __init__(self, model_dir: str | None = None, device: str | None = None):
        self.model_dir = Path(model_dir or DEFAULT_MODEL_DIR)
        self.device = device or ("cuda" if torch.cuda.is_available() else "cpu")
        self._model = None
        self._tokenizer = None
        self._id2label: dict[int, str] = {}
        self._loaded = False

    def load_model(self) -> None:
        if self._loaded:
            return
        if not self.model_dir.exists():
            raise FileNotFoundError(
                f"Checkpoint không tồn tại: {self.model_dir}. "
                "Chạy scripts/train_deberta_extractor.py trước."
            )
        logger.info(f"[DeBERTaExtractor] Loading from {self.model_dir} → {self.device}")
        self._tokenizer = AutoTokenizer.from_pretrained(str(self.model_dir))
        self._model = AutoModelForTokenClassification.from_pretrained(str(self.model_dir))
        self._model.to(self.device)
        self._model.eval()

        label_maps_path = self.model_dir / "label_maps.json"
        if label_maps_path.exists():
            with open(label_maps_path, encoding="utf-8") as f:
                maps = json.load(f)
            self._id2label = {int(k): v for k, v in maps["id2label"].items()}
        else:
            self._id2label = {int(k): v for k, v in self._model.config.id2label.items()}
        self._loaded = True
        logger.info(f"[DeBERTaExtractor] Ready. Labels: {list(self._id2label.values())}")

    async def extract(self, text: str, source: str = "User Story") -> list[ExtractedEntity]:
        if not text or not text.strip():
            return []
        if not self._loaded:
            self.load_model()

        logger.info(f"[DeBERTaExtractor] Extracting: {text[:80]}...")
        encoded = self._tokenizer(
            text,
            return_tensors="pt",
            truncation=True,
            max_length=128,
            return_offsets_mapping=True,
        )
        offset_mapping = encoded.pop("offset_mapping")[0].tolist()
        encoded = {k: v.to(self.device) for k, v in encoded.items()}

        with torch.no_grad():
            outputs = self._model(**encoded)
            predictions = outputs.logits.argmax(dim=-1)[0].cpu().tolist()

        entities_raw = self._bio_to_spans(text, predictions, offset_mapping)
        entity = self._spans_to_entity(entities_raw, text, source)
        return [entity] if entity else []

    def _bio_to_spans(
        self,
        text: str,
        pred_ids: list[int],
        offset_mapping: list[tuple[int, int]],
    ) -> dict[str, str]:
        spans: dict[str, list[str]] = {}
        current_type: str | None = None
        current_pieces: list[str] = []

        def flush():
            nonlocal current_type, current_pieces
            if current_type and current_pieces:
                phrase = " ".join(current_pieces).strip()
                spans.setdefault(current_type, []).append(phrase)
            current_type = None
            current_pieces = []

        for pred_id, (start, end) in zip(pred_ids, offset_mapping):
            if start == end:
                continue
            label = self._id2label.get(pred_id, "O")
            piece = text[start:end]
            if label == "O":
                flush()
                continue
            if label.startswith("B-"):
                flush()
                current_type = label[2:]
                current_pieces = [piece]
            elif label.startswith("I-"):
                tag_type = label[2:]
                if current_type == tag_type:
                    current_pieces.append(piece)
                else:
                    flush()
                    current_type = tag_type
                    current_pieces = [piece]
            else:
                flush()
        flush()
        return {k: (v[0] if len(v) == 1 else " ".join(v)) for k, v in spans.items()}

    def _spans_to_entity(
        self,
        spans: dict[str, str],
        text: str,
        source: str,
    ) -> ExtractedEntity | None:
        if not spans:
            logger.warning("[DeBERTaExtractor] Không trích được entity nào")
            return None

        role = spans.get("ROLE")
        action = spans.get("ACTION")
        resource = spans.get("RESOURCE")
        scope = spans.get("SCOPE")
        condition = spans.get("CONDITION")

        text_lower = text.lower()
        if "EFFECT" in spans:
            eff = spans["EFFECT"].lower()
            effect = (
                EffectType.DENY
                if any(w in eff for w in ("not", "cannot", "deny", "forbidden"))
                else EffectType.ALLOW
            )
        else:
            effect = (
                EffectType.DENY
                if any(w in text_lower for w in ("must not", "cannot", "should not", "may not"))
                else EffectType.ALLOW
            )

        # --- Hậu xử lý làm sạch span ---
        if action:
            action = action.lower().strip().split()[0]

        if role:
            role = role.strip().title()

        if resource:
            resource = resource.strip()
            for sw in ("assigned", "own", "all", "my", "their"):
                resource = resource.replace(sw.title(), "").replace(sw, "")
            resource = " ".join(resource.split()).title()

        if scope:
            scope_l = scope.lower()
            if "assigned" in scope_l:
                scope = "assigned"
            elif "own" in scope_l or "my" in scope_l:
                scope = "own"
            elif "all" in scope_l:
                scope = "all"
            else:
                scope = scope_l.split()[0]

        artifact = (
            ArtifactSource(source)
            if source in ArtifactSource._value2member_map_
            else ArtifactSource.USER_STORY
        )
        conf = 0.75 if (role and action and resource) else 0.5

        return ExtractedEntity(
            role=role,
            action=action,
            resource=resource,
            effect=effect,
            condition=condition,
            scope=scope,
            source=artifact,
            confidence=conf,
        )