<script setup>
import { reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useContainer } from "../../../application/container.js";
import { useErrorText } from "../../composables/i18n.js";
import { showToast } from "../../composables/ui.js";

const props = defineProps({ project: { type: Object, required: true } });
const emit = defineEmits(["changed"]);
const { t } = useI18n();
const { api } = useContainer();
const errorText = useErrorText();

const form = reactive({ name: "", description: "" });
watch(
  () => props.project,
  (p) => Object.assign(form, { name: p.name, description: p.description || "" }),
  { immediate: true }
);

async function save() {
  try {
    await api.projects.update(props.project.id, { name: form.name.trim(), description: form.description.trim() || null });
    showToast(t("settings.saved"), "success");
    emit("changed");
  } catch (e) {
    showToast(errorText(e), "error");
  }
}
</script>

<template>
  <div class="card">
    <h2>{{ t("settings.title") }}</h2>
    <div class="stack">
      <div>
        <label class="label">{{ t("projects.name") }}</label>
        <input v-model="form.name" class="field" />
      </div>
      <div>
        <label class="label">{{ t("settings.description") }}</label>
        <input v-model="form.description" class="field" />
      </div>
      <button type="button" class="btn btn-primary" @click="save">{{ t("common.saveChanges") }}</button>
    </div>
  </div>
</template>
