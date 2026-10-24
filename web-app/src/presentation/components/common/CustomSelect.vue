<script setup>
// Dropdown theo CSS gốc (.custom-select). Menu được teleport ra <body> với vị trí cố định
// nên không bị cắt khi nằm trong bảng cuộn ngang hay modal. Hỗ trợ bàn phím ↑ ↓ Enter Esc.
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";

const props = defineProps({
  modelValue: { type: [String, null], default: "" },
  options: { type: Array, required: true }, // [{ value, label }]
  placeholder: { type: String, default: "" },
  disabled: { type: Boolean, default: false },
  ariaLabel: { type: String, default: undefined },
});
const emit = defineEmits(["update:modelValue"]);

const open = ref(false);
const highlighted = ref(-1);
const root = ref(null);
const menu = ref(null);
const position = ref({ top: 0, left: 0, width: 0 });

const selectedIndex = computed(() => props.options.findIndex((o) => o.value === (props.modelValue ?? "")));
const label = computed(() => props.options[selectedIndex.value]?.label ?? props.placeholder);

function place() {
  const r = root.value?.getBoundingClientRect();
  if (r) position.value = { top: r.bottom, left: r.left, width: r.width };
}

function show() {
  if (props.disabled) return;
  document.dispatchEvent(new CustomEvent("custom-select:close-all"));
  place();
  highlighted.value = Math.max(selectedIndex.value, 0);
  open.value = true;
  nextTick(scrollToHighlighted);
}
function hide() {
  open.value = false;
}
const toggle = () => (open.value ? hide() : show());

function choose(value) {
  if (value !== props.modelValue) emit("update:modelValue", value);
  hide();
}

function scrollToHighlighted() {
  menu.value?.children[highlighted.value]?.scrollIntoView({ block: "nearest" });
}

function onKeydown(e) {
  if (props.disabled) return;
  if (!open.value && ["ArrowDown", "ArrowUp", "Enter", " "].includes(e.key)) {
    e.preventDefault();
    show();
    return;
  }
  if (!open.value) return;
  const n = props.options.length;
  if (e.key === "ArrowDown" || e.key === "ArrowUp") {
    e.preventDefault();
    highlighted.value = (highlighted.value + (e.key === "ArrowDown" ? 1 : -1) + n) % n;
    scrollToHighlighted();
  } else if (e.key === "Enter") {
    e.preventDefault();
    const option = props.options[highlighted.value];
    if (option) choose(option.value);
  } else if (e.key === "Escape" || e.key === "Tab") {
    e.stopPropagation();
    hide();
  }
}

function onDocumentClick(e) {
  if (!root.value?.contains(e.target) && !menu.value?.contains(e.target)) hide();
}

// Chỉ gắn listener khi menu đang mở
watch(open, (isOpen) => {
  const method = isOpen ? "addEventListener" : "removeEventListener";
  document[method]("click", onDocumentClick, true);
  document[method]("custom-select:close-all", hide);
  window[method]("resize", hide);
  window[method]("scroll", place, true);
});
onBeforeUnmount(() => hide());
</script>

<template>
  <div ref="root" class="custom-select" :class="{ open, 'is-disabled': disabled }" :data-value="modelValue">
    <button
      type="button"
      class="custom-select-trigger"
      aria-haspopup="listbox"
      :aria-expanded="open"
      :aria-label="ariaLabel"
      :disabled="disabled"
      @click="toggle"
      @keydown="onKeydown"
    >
      {{ label }}
    </button>
    <Teleport to="body">
      <div
        v-if="open"
        class="custom-select open custom-select-floating"
        :style="{ top: `${position.top}px`, left: `${position.left}px`, width: `${position.width}px` }"
      >
        <div ref="menu" class="custom-select-menu" role="listbox">
          <div v-if="!options.length" class="custom-select-option placeholder">{{ placeholder }}</div>
          <div
            v-for="(o, i) in options"
            :key="o.value"
            class="custom-select-option"
            :class="{ selected: o.value === (modelValue ?? ''), highlighted: i === highlighted }"
            role="option"
            :aria-selected="o.value === (modelValue ?? '')"
            @mouseenter="highlighted = i"
            @click="choose(o.value)"
          >
            {{ o.label }}
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
