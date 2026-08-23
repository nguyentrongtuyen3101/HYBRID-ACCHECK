"""
Debug script — chạy trực tiếp trên máy (trong venv, từ thư mục gốc project):
    python debug_case.py

In ra:
1. Extractor có nhận diện đúng Role/Action/Resource không
2. Embedding cosine similarity thực tế giữa entity extract được và PM row
3. Kết luận vì sao bị đoán sai
"""
import asyncio
import sys
sys.path.insert(0, ".")

from app.services.extractor.rule_based_extractor import RuleBasedExtractor
from app.services.normalizer.sbert_normalizer import SBERTNormalizer
from app.core.config import get_settings


async def main():
    settings = get_settings()
    print(f"ALIGNMENT_THRESHOLD hiện tại: {settings.ALIGNMENT_THRESHOLD}")
    print(f"Weights: role={settings.ROLE_WEIGHT}, action={settings.ACTION_WEIGHT}, resource={settings.RESOURCE_WEIGHT}")
    print()

    extractor = RuleBasedExtractor()
    normalizer = SBERTNormalizer()

    # Case AC_077 — theo dataset, PM là role="Localization Manager", action="approve", resource="Translation"
    test_text = "As a Localization Manager, I want to approve translation, so that I can ensure translation quality."
    pm_role, pm_action, pm_resource = "Localization Manager", "approve", "Translation"

    entities = await extractor.extract(test_text)
    print("=== EXTRACTOR OUTPUT ===")
    if not entities:
        print("KHÔNG extract được entity nào! (đây là nguyên nhân nếu xảy ra)")
        return
    e = entities[0]
    print(f"Role extracted:     '{e.role}'")
    print(f"Action extracted:   '{e.action}'")
    print(f"Resource extracted: '{e.resource}'")
    print()
    print(f"PM Role:     '{pm_role}'")
    print(f"PM Action:   '{pm_action}'")
    print(f"PM Resource: '{pm_resource}'")
    print()

    print("=== SBERT SIMILARITY ===")
    role_emb_e = await normalizer.embed_text(e.role)
    role_emb_pm = await normalizer.embed_text(pm_role)
    action_emb_e = await normalizer.embed_text(e.action)
    action_emb_pm = await normalizer.embed_text(pm_action)
    resource_emb_e = await normalizer.embed_text(e.resource)
    resource_emb_pm = await normalizer.embed_text(pm_resource)

    role_sim = normalizer.cosine_similarity(role_emb_e, role_emb_pm)
    action_sim = normalizer.cosine_similarity(action_emb_e, action_emb_pm)
    resource_sim = normalizer.cosine_similarity(resource_emb_e, resource_emb_pm)

    print(f"Role similarity:     {role_sim:.4f}")
    print(f"Action similarity:   {action_sim:.4f}")
    print(f"Resource similarity: {resource_sim:.4f}")

    score = (
        settings.ROLE_WEIGHT * role_sim
        + settings.ACTION_WEIGHT * action_sim
        + settings.RESOURCE_WEIGHT * resource_sim
    )
    print(f"\nWeighted score: {score:.4f}")
    print(f"Threshold:      {settings.ALIGNMENT_THRESHOLD}")
    print(f"=> {'MATCHED (Normal)' if score >= settings.ALIGNMENT_THRESHOLD else 'KHÔNG MATCH (Missing Permission)'}")


if __name__ == "__main__":
    asyncio.run(main())