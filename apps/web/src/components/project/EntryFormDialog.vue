<script setup lang="ts">
import { computed, reactive, ref, watch, onBeforeUnmount, nextTick } from 'vue';
import { onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Select as UiSelect, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/ui/select';
import { saveEntry, type PersonalEntry, type EntryInput } from '@/api/entries';
const props = defineProps<{ open:boolean; entry?:PersonalEntry|null; compact?:boolean }>();
const emit = defineEmits<{ 'update:open':[value:boolean]; saved:[entry:PersonalEntry,addAccount:boolean] }>();
const empty = ():EntryInput => ({source:'private',targetId:0,kind:'tool',name:'',url:'',groupName:'',pinned:false,note:'',revision:0});
const form = reactive(empty());
const busy=ref(false), error=ref(''), addAccount=ref(false), baseline=ref('');
const discardOpen=ref(false);let pending:((value:boolean)=>void)|null=null;
const dirty=computed(()=>props.open && JSON.stringify(form)!==baseline.value);
watch(()=>props.open,open=>{if(open){Object.assign(form,empty(),props.entry||{});baseline.value=JSON.stringify(form);error.value='';addAccount.value=false}}, {immediate:true});
function askDiscard():Promise<boolean>{if(busy.value)return Promise.resolve(false);if(!dirty.value)return Promise.resolve(true);discardOpen.value=true;return new Promise(resolve=>{pending=resolve})}
function resolveDiscard(value:boolean){discardOpen.value=false;pending?.(value);pending=null}
async function close(){if(await askDiscard())emit('update:open',false)}
const opened=computed({get:()=>props.open,set:value=>{if(!value)void close()}});
onBeforeRouteLeave(()=>props.compact?true:askDiscard());onBeforeRouteUpdate(()=>props.compact?true:askDiscard());
defineExpose({confirmLeave:askDiscard});
function beforeUnload(event:BeforeUnloadEvent){if(dirty.value){event.preventDefault();event.returnValue=''}}
window.addEventListener('beforeunload',beforeUnload);onBeforeUnmount(()=>{window.removeEventListener('beforeunload',beforeUnload);pending?.(false);form.note=''});
function urlValid(){try{const u=new URL(form.url);return ['https:','http:'].includes(u.protocol)&&!!u.hostname&&!u.username&&!u.password}catch{return false}}
async function submit(){
 if(busy.value)return;
 if(!form.name.trim()||(form.source==='private'&&!urlValid())){error.value='请填写名称和有效的 HTTP 或 HTTPS 网址，网址中不要包含账号密码。';await nextTick();document.querySelector<HTMLElement>('[data-entry-form] [aria-invalid="true"]')?.focus();return}
 busy.value=true;error.value='';
 try{const result=await saveEntry(props.entry?.id||null,{...form});baseline.value=JSON.stringify(form);emit('update:open',false);emit('saved',result.data,addAccount.value)}catch(e){error.value=e instanceof Error?e.message:'保存失败，请重试'}finally{busy.value=false}
}
</script>
<template>
 <Dialog v-model:open="opened"><DialogContent class="max-h-[calc(100dvh-2rem)] overflow-y-auto" @escape-key-down="busy && $event.preventDefault()">
  <DialogHeader><DialogTitle>{{ entry ? '编辑入口' : '添加私人入口' }}</DialogTitle><DialogDescription>仅本人可见。名称和网址即可保存，账号可以稍后添加。</DialogDescription></DialogHeader>
  <form data-entry-form novalidate class="grid gap-4" @submit.prevent="submit">
   <p v-if="error" id="entry-form-error" role="alert" class="notice notice-error">{{ error }}</p>
   <fieldset :disabled="busy" class="grid min-w-0 gap-4">
    <label class="ui-field"><span>名称 *</span><Input v-model="form.name" :readonly="form.source!=='private'" :aria-invalid="!!error&&!form.name.trim()" :aria-describedby="error?'entry-form-error':undefined" placeholder="例如：我的测试环境" maxlength="200" /></label>
    <label class="ui-field"><span>网址 *</span><Input v-model="form.url" :readonly="form.source!=='private'" :aria-invalid="!!error&&form.source==='private'&&!urlValid()" :aria-describedby="error?'entry-form-error':undefined" placeholder="https://example.com" maxlength="1000" /></label>
    <div v-if="form.source==='private'" class="ui-field"><span id="entry-kind-label">类型</span><UiSelect v-model="form.kind"><SelectTrigger aria-labelledby="entry-kind-label"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="tool">工具</SelectItem><SelectItem value="project">项目</SelectItem></SelectContent></UiSelect></div>
    <label class="ui-field"><span>分组（选填）</span><Input v-model="form.groupName" placeholder="例如：日常工作、个人项目" maxlength="80" /></label>
    <label class="ui-field"><span>私人备注（选填）</span><Textarea v-model="form.note" class="resize-none" placeholder="仅本人可见的入口说明" maxlength="10000" /></label>
    <div class="flex items-center justify-between gap-4"><label for="entry-pinned">置顶</label><Switch id="entry-pinned" v-model="form.pinned" /></div>
    <div v-if="!entry && !compact" class="flex items-center justify-between gap-4"><label for="entry-add-account">保存后添加账号</label><Switch id="entry-add-account" v-model="addAccount" /></div>
   </fieldset>
   <DialogFooter><Button type="button" variant="outline" :disabled="busy" @click="close">取消</Button><Button type="submit" :disabled="busy">{{ busy?'正在保存…':'保存入口' }}</Button></DialogFooter>
  </form>
 </DialogContent></Dialog>
 <Dialog :open="discardOpen" @update:open="!$event&&resolveDiscard(false)"><DialogContent><DialogHeader><DialogTitle>放弃入口修改？</DialogTitle><DialogDescription>未保存的名称、网址和备注将被清除。</DialogDescription></DialogHeader><DialogFooter><Button variant="outline" @click="resolveDiscard(false)">继续编辑</Button><Button variant="destructive" @click="resolveDiscard(true)">放弃修改</Button></DialogFooter></DialogContent></Dialog>
</template>
