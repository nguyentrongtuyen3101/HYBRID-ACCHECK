"""
Requirement Extractor - phiên bản Rule-based (pattern matching).

LÝ DO CÓ FILE NÀY:
Theo tài liệu đề cương, Requirement Extractor chuẩn phải dùng DeBERTa
Token Classification (fine-tuned trên data đã gán nhãn Role/Action/Resource/
Effect/Condition/Scope). Tuy nhiên việc fine-tune DeBERTa đòi hỏi:
  - Bộ data đã gán nhãn ở mức token (chưa có, phải tự làm)
  - GPU + thời gian train
Trong phạm vi 2 tuần đầu, class này đóng vai trò "cắm tạm" để cả pipeline
chạy được end-to-end với dữ liệu thật, đúng nguyên tắc Clean Architecture
của project (mỗi service có base.py interface + implementation cụ thể →
dễ thay thế). Khi có đủ data + thời gian, chỉ cần thay
`extractor = RuleBasedExtractor()` bằng `extractor = DeBERTaExtractor()`
trong pipeline.py, không phải sửa gì khác.

CÁCH HOẠT ĐỘNG:
Dùng regex bám theo mẫu câu Connextra ("As a <role>, I want to <action>
<resource> ...") và các từ khóa nghiệp vụ phổ biến để suy ra Effect
(Allow/Deny) và Scope (assigned/own/all...).
"""
import re

from loguru import logger

from app.domain.enums.conflict import ArtifactSource, EffectType
from app.domain.entities.access_control import ExtractedEntity
from app.services.extractor.base import BaseExtractor

# Mẫu câu Connextra chuẩn: "As a/an <role>, I want to/can <action> <resource>"
_CONNEXTRA_PATTERN = re.compile(
    r"as\s+an?\s+(?P<role>[\w\s\-]+?),\s*i\s+(?:want|need|would like)\s+to\s+"
    r"(?P<action>[\w\-]+)\s+(?P<resource>[\w\s\-]+?)"
    r"(?:,?\s*so\s+that|\.|$)",
    re.IGNORECASE,
)

# Mẫu câu dạng đơn giản: "<Role> may/can/should <action> <resource>"
_SIMPLE_PATTERN = re.compile(
    r"(?P<role>[\w\s\-]+?)\s+(?:may|can|shall|should be able to|is able to)\s+"
    r"(?P<action>[\w\-]+)\s+(?P<resource>[\w\s\-]+?)"
    r"(?:,|\.|$)",
    re.IGNORECASE,
)

_DENY_KEYWORDS = ("must not", "cannot", "can not", "should not", "may not", "is not allowed to")

_SCOPE_KEYWORDS = {
    "assigned": ["assigned", "their own", "belongs to them", "of their own"],
    "own": ["own ", "themselves", "their profile"],
    "all": ["all ", "any ", "every "],
}

_ACTION_VERBS = {"view", "read", "edit", "update", "create", "delete", "remove", "add", "approve", "reject", "manage"}

# Từ đồng nghĩa quy về động từ chuẩn, để so khớp tốt hơn với Action trong Permission Matrix
_ACTION_SYNONYMS = {
    "generate": "create", "add": "create", "register": "create", "build": "create",
    "see": "view", "access": "view", "find": "view", "check": "view", "monitor": "view", "look": "view",
    "modify": "edit", "customise": "edit", "customize": "edit", "set": "edit", "maintain": "edit", "update": "edit",
    "remove": "delete", "cancel": "delete",
    "grant": "approve", "authorise": "approve", "authorize": "approve", "confirm": "approve",
    "decline": "reject",
    "release": "publish",
    "allocate": "assign",
    "download": "export",
    "filter": "search",
    "administer": "manage",
}

# Các cụm từ nối mục đích/điều kiện — Resource phải dừng lại NGAY TRƯỚC các cụm này,
# vì phần sau thường là mệnh đề phụ (purpose clause), không phải một phần của Resource.
_PURPOSE_BOUNDARY = re.compile(
    r"\s+(?:to\s+\w+|for\s+\w+|via\s+\w+|by\s+\w+|in\s+order\s+to|which\s+is|that\s+is|"
    r"so\s+(?:that|I)|within\s+the|from\s+the).*$",
    re.IGNORECASE,
)


class RuleBasedExtractor(BaseExtractor):
    """Extractor tạm thời dùng pattern-matching, thay cho DeBERTa trong giai đoạn đầu."""

    async def extract(self, text: str, source: str = "User Story") -> list[ExtractedEntity]:
        logger.info(f"[Extractor] (rule-based) Extracting from: {text[:80]}...")
        if not text or not text.strip():
            return []

        text_lower = text.lower()
        artifact_source = (
            ArtifactSource(source) if source in ArtifactSource._value2member_map_ else ArtifactSource.USER_STORY
        )

        match = _CONNEXTRA_PATTERN.search(text) or _SIMPLE_PATTERN.search(text)
        if not match:
            logger.warning(f"[Extractor] Không khớp pattern nào cho: {text[:80]}")
            return []

        role = match.group("role").strip().title()
        action_raw = match.group("action").strip().lower()
        resource_raw = match.group("resource").strip().rstrip(".")

        # Chuẩn hóa action: ưu tiên tra từ đồng nghĩa trước (để "add"->"create",
        # "download"->"export"...), nếu không có trong từ điển mới xét khớp trực
        # tiếp với danh sách action chuẩn.
        action = _ACTION_SYNONYMS.get(
            action_raw, next((v for v in _ACTION_VERBS if v == action_raw or v in action_raw), action_raw)
        )

        effect = EffectType.DENY if any(kw in text_lower for kw in _DENY_KEYWORDS) else EffectType.ALLOW

        scope = None
        for scope_label, keywords in _SCOPE_KEYWORDS.items():
            if any(kw in text_lower for kw in keywords):
                scope = scope_label
                break

        # Bóc tách các từ khóa scope (assigned/own/all/any/every) ra khỏi Resource
        # để Resource chỉ còn danh từ thuần túy (vd "assigned customers" → "customers")
        resource_clean = resource_raw
        for keywords in _SCOPE_KEYWORDS.values():
            for kw in keywords:
                kw_stripped = kw.strip()
                pattern = re.compile(rf"^\s*{re.escape(kw_stripped)}\s*", re.IGNORECASE)
                resource_clean = pattern.sub("", resource_clean)

        # Cắt bỏ mệnh đề mục đích/điều kiện phía sau (vd "voucher codes TO GRANT DISCOUNTS..."
        # -> chỉ giữ "voucher codes"), để Resource không bị lố quá dài khi so khớp semantic
        resource_clean = _PURPOSE_BOUNDARY.sub("", resource_clean)

        resource = resource_clean.strip().title() or resource_raw.title()

        entity = ExtractedEntity(
            role=role,
            action=action,
            resource=resource,
            effect=effect,
            condition=None,
            scope=scope,
            source=artifact_source,
            confidence=0.6,  # rule-based nên confidence thấp hơn model đã train
        )
        return [entity]