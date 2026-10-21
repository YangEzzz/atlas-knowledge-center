<script setup lang="ts">
import {
  computed,
  reactive,
  ref,
  onMounted,
  onBeforeUnmount,
  watch,
} from "vue";
import { toast } from "vue-sonner";
import {
  documentTypes,
  saveTemplate,
  type DocumentTemplate,
} from "@/api/management";
import RichTextField from "@/components/app/RichTextField.vue";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select as UiSelect,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { stripHtml } from "@/utils/markdown";
const props = defineProps<{ row?: DocumentTemplate }>();
const emit = defineEmits<{ close: []; saved: [] }>();
const form = reactive({
  name: props.row?.name || "",
  type: props.row?.type || "environment",
  content: props.row?.content || "",
  isEnabled: props.row?.isEnabled || false,
});
const initial = JSON.stringify(form);
const dirty = computed(() => JSON.stringify(form) !== initial);
const saving = ref(false);
const error = ref("");
const invalidField = ref("");
const discard = ref(false);
const nameInput = ref<InstanceType<typeof Input>>();
let leaveResolver: ((value: boolean) => void) | undefined;
function confirmLeave(): Promise<boolean> {
  if (saving.value) return Promise.resolve(false);
  if (!dirty.value) return Promise.resolve(true);
  discard.value = true;
  return new Promise((resolve) => {
    leaveResolver = resolve;
  });
}
function finishDiscard() {
  const resolve = leaveResolver;
  leaveResolver = undefined;
  resolve?.(true);
  emit("close");
}
watch(discard, (value) => {
  if (!value) {
    leaveResolver?.(false);
    leaveResolver = undefined;
  }
});
defineExpose({ confirmLeave });
function close() {
  if (saving.value) return;
  if (dirty.value) discard.value = true;
  else emit("close");
}
function unload(event: BeforeUnloadEvent) {
  if (dirty.value) {
    event.preventDefault();
    event.returnValue = "";
  }
}
onMounted(() => window.addEventListener("beforeunload", unload));
onBeforeUnmount(() => window.removeEventListener("beforeunload", unload));
async function save() {
  if (saving.value) return;
  invalidField.value = "";
  error.value = "";
  if (!form.name.trim()) {
    invalidField.value = "name";
    error.value = "请填写模板名称";
    (nameInput.value?.$el as HTMLInputElement)?.focus();
    return;
  }
  if (!stripHtml(form.content)) {
    invalidField.value = "content";
    error.value = "请填写模板正文";
    document.querySelector<HTMLElement>("[data-slate-editor]")?.focus();
    return;
  }
  saving.value = true;
  try {
    await saveTemplate(props.row?.id, {
      ...form,
      name: form.name.trim(),
      updatedAt: props.row?.updatedAt,
    });
    toast.success("模板已保存");
    emit("saved");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "保存失败，请重试";
  } finally {
    saving.value = false;
  }
}
</script>
<template>
  <Dialog
    :open="true"
    @update:open="
      (value) => {
        if (!value) close();
      }
    "
  >
    <DialogContent
      class="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-4xl"
      @interact-outside="
        (event) => {
          if (saving || discard) event.preventDefault();
        }
      "
      @escape-key-down="
        (event) => {
          if (saving || discard) event.preventDefault();
        }
      "
    >
      <DialogHeader>
        <DialogTitle>{{ row ? "编辑文档模板" : "新增文档模板" }}</DialogTitle>
        <DialogDescription>
          启用后用于同类型新文档；同类型的其他启用模板会自动停用。已保存文档不受影响。
        </DialogDescription>
      </DialogHeader>
      <form novalidate class="space-y-5" @submit.prevent="save">
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="ui-field">
            <label for="template-name">模板名称</label>
            <Input
              id="template-name"
              ref="nameInput"
              v-model="form.name"
              maxlength="200"
              :disabled="saving"
              :aria-invalid="invalidField === 'name'"
              :aria-describedby="error ? 'template-error' : undefined"
              placeholder="例如：后端服务部署说明"
            />
          </div>
          <div class="ui-field">
            <label for="template-type">文档类型</label>
            <UiSelect v-model="form.type" :disabled="saving">
              <SelectTrigger id="template-type"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="type in documentTypes"
                  :key="type.value"
                  :value="type.value"
                >
                  {{ type.label }}
                </SelectItem>
              </SelectContent>
            </UiSelect>
          </div>
        </div>
        <div class="ui-field">
          <span id="template-content-label">模板正文</span>
          <div
            role="group"
            aria-labelledby="template-content-label"
            :aria-invalid="invalidField === 'content'"
            :aria-describedby="error ? 'template-error' : undefined"
          >
            <RichTextField
              v-model="form.content"
              :disabled="saving"
              :invalid="invalidField === 'content'"
              :described-by="error ? 'template-error' : undefined"
            />
          </div>
        </div>
        <div class="flex items-center gap-3">
          <Switch
            id="template-enabled"
            v-model="form.isEnabled"
            :disabled="saving"
          />
          <label for="template-enabled">启用此模板</label>
        </div>
        <p
          v-if="error"
          id="template-error"
          role="alert"
          class="text-sm text-destructive"
        >
          {{ error }}
        </p>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            :disabled="saving"
            @click="close"
          >
            取消
          </Button>
          <Button type="submit" :disabled="saving" class="min-w-24">
            {{ saving ? "保存中…" : "保存模板" }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
  <Dialog v-model:open="discard">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>放弃未保存的模板？</DialogTitle>
        <DialogDescription>
          本次填写的内容尚未保存，关闭后将丢失。
        </DialogDescription>
      </DialogHeader>
      <DialogFooter>
        <Button variant="outline" @click="discard = false">继续编辑</Button>
        <Button variant="destructive" @click="finishDiscard">放弃更改</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
