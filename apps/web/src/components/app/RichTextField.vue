<script setup lang="ts">
import "@wangeditor/editor/dist/css/style.css";
import { onMounted, onBeforeUnmount, ref, shallowRef, watch } from "vue";
import {
  createEditor,
  createToolbar,
  type IDomEditor,
  type Toolbar,
} from "@wangeditor/editor";
const props = defineProps<{
  modelValue: string;
  disabled?: boolean;
  invalid?: boolean;
  describedBy?: string;
}>();
const emit = defineEmits<{ "update:modelValue": [value: string] }>();
const body = ref<HTMLElement>();
const toolbar = ref<HTMLElement>();
const editor = shallowRef<IDomEditor>();
let bar: Toolbar | undefined;
let syncing = false;
onMounted(() => {
  editor.value = createEditor({
    selector: body.value!,
    html: props.modelValue || "<p><br></p>",
    config: {
      placeholder: "编写模板正文，支持标题、列表、表格和链接…",
      scroll: false,
      onChange(instance) {
        if (!syncing) emit("update:modelValue", instance.getHtml());
      },
    },
  });
  bar = createToolbar({
    editor: editor.value,
    selector: toolbar.value!,
    config: { excludeKeys: ["group-image", "group-video", "fullScreen"] },
  });
  if (props.disabled) editor.value.disable();
  updateAccessibility();
});
function updateAccessibility() {
  const element = body.value?.querySelector("[contenteditable]");
  element?.setAttribute("role", "textbox");
  element?.setAttribute("aria-label", "模板正文");
  element?.setAttribute("aria-multiline", "true");
  element?.setAttribute("aria-invalid", String(!!props.invalid));
  if (props.describedBy)
    element?.setAttribute("aria-describedby", props.describedBy);
  else element?.removeAttribute("aria-describedby");
}
watch(() => [props.invalid, props.describedBy], updateAccessibility);
watch(
  () => props.modelValue,
  (value) => {
    if (editor.value && editor.value.getHtml() !== value) {
      syncing = true;
      editor.value.setHtml(value || "<p><br></p>");
      syncing = false;
    }
  },
);
watch(
  () => props.disabled,
  (disabled) => (disabled ? editor.value?.disable() : editor.value?.enable()),
);
onBeforeUnmount(() => {
  bar?.destroy();
  editor.value?.destroy();
});
</script>
<template>
  <div class="wang-editor-shell">
    <div ref="toolbar" class="wang-editor-toolbar" />
    <div ref="body" class="wang-editor-body" />
  </div>
</template>
