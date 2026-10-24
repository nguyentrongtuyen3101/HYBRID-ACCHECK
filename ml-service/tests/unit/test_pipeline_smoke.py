import pytest
from accheck_ml.domain.schemas.access_control import (
    PipelineInput,
    PermissionMatrixRowSchema,
)
from accheck_ml.domain.enums.conflict import EffectType
from accheck_ml.services.pipeline import HybridACCheckPipeline


@pytest.mark.asyncio
async def test_pipeline_smoke():
    payload = PipelineInput(
        user_story="As a Sales Staff, I want to view assigned customers",
        acceptance_criteria=[],
        permission_matrix=[
            PermissionMatrixRowSchema(
                role="Sales Staff",
                action="View",
                resource="Customer",
                effect=EffectType.ALLOW,
                scope="Assigned",
            )
        ],
    )

    pipeline = HybridACCheckPipeline()
    result = await pipeline.run(payload)

    assert result is not None
    assert len(result.extracted_entities) >= 1
    assert len(result.conflicts) >= 1
    assert result.summary is not None
    print(result.summary)
