from typing import Any
import networkx as nx
from loguru import logger

from app.domain.entities.access_control import ExtractedEntity, PermissionMatrixEntry
from app.domain.enums.conflict import ArtifactSource
from app.services.aligner.base import AlignmentResult


class AccessControlRequirementGraph:
    def __init__(self):
        self.graph = nx.MultiDiGraph()

    def build_from_alignment(
        self,
        alignment_results: list[AlignmentResult],
        permission_matrix: list[PermissionMatrixEntry],
    ) -> None:
        logger.info("[ACRG] Building graph from alignment results")

        for ar in alignment_results:
            self._add_entity_nodes(ar.extracted)

        for row in permission_matrix:
            self._add_pm_row_nodes(row)

        for ar in alignment_results:
            if ar.is_matched and ar.matched_pm_row:
                self._add_alignment_edge(ar)

    def _add_entity_nodes(self, ent: ExtractedEntity) -> None:
        source = ent.source.value
        if ent.role:
            self.graph.add_node(f"role:{ent.role}", type="Role", value=ent.role, source=source)
        if ent.action:
            self.graph.add_node(f"action:{ent.action}", type="Action", value=ent.action, source=source)
        if ent.resource:
            self.graph.add_node(f"resource:{ent.resource}", type="Resource", value=ent.resource, source=source)
        if ent.role and ent.action:
            self.graph.add_edge(f"role:{ent.role}", f"action:{ent.action}", relation="performs", source=source)
        if ent.action and ent.resource:
            self.graph.add_edge(f"action:{ent.action}", f"resource:{ent.resource}", relation="on", source=source)

    def _add_pm_row_nodes(self, row: PermissionMatrixEntry) -> None:
        source = ArtifactSource.PERMISSION_MATRIX.value
        self.graph.add_node(f"role:{row.role}", type="Role", value=row.role, source=source)
        self.graph.add_node(f"action:{row.action}", type="Action", value=row.action, source=source)
        self.graph.add_node(f"resource:{row.resource}", type="Resource", value=row.resource, source=source)
        self.graph.add_edge(
            f"role:{row.role}",
            f"action:{row.action}",
            relation="performs",
            source=source,
            effect=row.effect.value,
            condition=row.condition,
            scope=row.scope,
        )

    def _add_alignment_edge(self, ar: AlignmentResult) -> None:
        ent = ar.extracted
        if ent.role and ar.matched_pm_row:
            self.graph.add_edge(
                f"role:{ent.role}",
                f"role:{ar.matched_pm_row.role}",
                relation="aligned_with",
                score=ar.similarity_score,
            )

    def to_dict(self) -> dict[str, Any]:
        return {
            "nodes": list(self.graph.nodes(data=True)),
            "edges": list(self.graph.edges(data=True)),
        }
