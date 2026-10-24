<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useEnumLabel } from "../../composables/i18n.js";

const props = defineProps({ diff: { type: Object, required: true } });
const { t } = useI18n();
const label = useEnumLabel();
const empty = computed(() => !props.diff.added.length && !props.diff.removed.length && !props.diff.changed.length);
const describe = (r) => `${r.role} · ${r.action} · ${r.resource}`;
const detail = (r) =>
  [label("effect", r.effect), r.scope && t("diff.scope", { scope: r.scope }), r.condition && t("diff.condition", { condition: r.condition })]
    .filter(Boolean)
    .join(", ");
</script>

<template>
  <div class="stack">
    <div v-if="empty" class="muted">{{ t("diff.none") }}</div>
    <div v-for="r in diff.added" :key="'a' + describe(r)" class="diff-row diff-add">
      <strong>+ {{ describe(r) }}</strong> <span class="muted">{{ detail(r) }}</span>
    </div>
    <div v-for="r in diff.removed" :key="'r' + describe(r)" class="diff-row diff-del">
      <strong>− {{ describe(r) }}</strong> <span class="muted">{{ detail(r) }}</span>
    </div>
    <div v-for="c in diff.changed" :key="'c' + describe(c.after)" class="diff-row diff-chg">
      <strong>~ {{ describe(c.after) }}</strong>
      <span class="muted">{{ detail(c.before) }} → {{ detail(c.after) }}</span>
    </div>
  </div>
</template>
