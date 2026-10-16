<script setup lang="ts">
import { ref, watch, onBeforeUnmount, nextTick } from 'vue';
import { useRouter } from 'vue-router';
import { Plus, ExternalLink, Copy, KeyRound, Pin, PencilLine, Trash2, Link2 } from 'lucide-vue-next';
import { toast } from 'vue-sonner';
import { Button, buttonVariants } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select as UiSelect, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import PageHeader from '@/components/ui/PageHeader.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import ProjectIdentity from '@/components/project/ProjectIdentity.vue';
import EntryFormDialog from '@/components/project/EntryFormDialog.vue';
import { listEntries, saveEntry, deleteEntry, restoreEntry, type PersonalEntry } from '@/api/entries';
import { usePersonalRecords } from '@/composables/use-personal-records';
const router=useRouter();const {syncFavorites}=usePersonalRecords();
const rows=ref<PersonalEntry[]>([]),total=ref(0),page=ref(1),keyword=ref(''),source=ref('all'),group=ref(''),trash=ref(false),composing=ref(false);
const loading=ref(false),error=ref(''),busy=ref(false),editorOpen=ref(false),editing=ref<PersonalEntry|null>(null),deleting=ref<PersonalEntry|null>(null),deleteError=ref('');
let epoch=0,timer:ReturnType<typeof setTimeout>|undefined;
async function load(){const ticket=++epoch;loading.value=true;error.value='';try{await syncFavorites();const result=await listEntries({page:page.value,pageSize:24,keyword:keyword.value,source:source.value==='all'?undefined:source.value,group:group.value,trash:trash.value});if(ticket!==epoch)return;rows.value=result.data.list;total.value=result.data.total;if(page.value>1&&!rows.value.length){page.value--;return}}catch(e){if(ticket===epoch)error.value=e instanceof Error?e.message:'入口无法加载，请重试'}finally{if(ticket===epoch)loading.value=false}}
watch([keyword,group,composing],()=>{clearTimeout(timer);if(composing.value)return;timer=setTimeout(()=>{if(page.value!==1)page.value=1;else void load()},300)});
watch([source,trash],()=>{rows.value=[];total.value=0;if(page.value!==1)page.value=1;else void load()});watch(page,()=>void load());void load();onBeforeUnmount(()=>{epoch++;clearTimeout(timer);rows.value=[]});
function create(){editing.value=null;editorOpen.value=true}
function edit(entry:PersonalEntry){editing.value=entry;editorOpen.value=true}
function accountPath(entry:PersonalEntry){return {path:'/my-accounts',query:entry.source==='private'?{entryId:entry.id}:entry.source==='tool'?{toolId:entry.targetId}:{projectId:entry.targetId}}}
async function saved(entry:PersonalEntry,addAccount:boolean){toast.success('入口已保存');await syncFavorites(true);await load();if(addAccount)await router.push({...accountPath(entry),query:{...accountPath(entry).query,create:'1'}})}
async function pin(entry:PersonalEntry){if(busy.value)return;busy.value=true;try{await saveEntry(entry.id,{...entry,pinned:!entry.pinned});await load()}catch(e){toast.error(e instanceof Error?e.message:'置顶失败')}finally{busy.value=false}}
async function clearFilter(kind:'keyword'|'group'){if(kind==='keyword')keyword.value='';else group.value='';await nextTick();document.querySelector<HTMLInputElement>(kind==='keyword'?'input[aria-label="搜索我的入口"]':'input[aria-label="筛选入口分组"]')?.focus()}
async function copy(entry:PersonalEntry){try{await navigator.clipboard.writeText(entry.url.startsWith('/')?new URL(entry.url,location.origin).href:entry.url);toast.success('链接已复制')}catch{toast.error('复制失败，请手动复制链接')}}
function open(entry:PersonalEntry){if(entry.url.startsWith('/'))void router.push(entry.url);else if(/^https?:\/\//i.test(entry.url))window.open(entry.url,'_blank','noopener,noreferrer')}
async function remove(){if(!deleting.value||busy.value)return;busy.value=true;deleteError.value='';try{await deleteEntry(deleting.value);deleting.value=null;await syncFavorites(true);await load();toast.success('入口已移入回收站')}catch(e){deleteError.value=e instanceof Error?e.message:'删除失败，请重试'}finally{busy.value=false}}
async function unfavorite(entry:PersonalEntry){if(busy.value)return;busy.value=true;try{await deleteEntry(entry);await syncFavorites(true);await load()}catch(e){toast.error(e instanceof Error?e.message:'取消收藏失败')}finally{busy.value=false}}
async function restore(entry:PersonalEntry){if(busy.value)return;busy.value=true;try{await restoreEntry(entry.id);await load();toast.success('入口及关联账号已恢复')}catch(e){toast.error(e instanceof Error?e.message:'恢复失败')}finally{busy.value=false}}
const label=(entry:PersonalEntry)=>entry.source==='private'?(entry.kind==='project'?'私人项目':'私人工具'):entry.source==='project'?'公司项目':entry.source==='tool'?'公司工具':'文档收藏';
</script>
<template>
 <PageHeader title="我的入口" description="收藏公司内容，保存自己的项目和工具。分组、备注与账号仅本人可见。"><template #actions><Button variant="outline" :disabled="busy||loading" @click="trash=!trash">{{trash?'返回入口':'回收站'}}</Button><Button v-if="!trash" @click="create"><Plus class="size-4" />添加入口</Button></template></PageHeader>
 <div class="mb-6 grid gap-3 sm:grid-cols-[minmax(0,1fr)_160px_200px]">
  <div class="relative"><Input v-model="keyword" class="pr-16" aria-label="搜索我的入口" placeholder="搜索名称、网址或分组" @compositionstart="composing=true" @compositionend="composing=false" /><Button v-if="keyword" variant="ghost" size="sm" class="absolute right-0 top-0" aria-label="清空入口搜索" @click="clearFilter('keyword')">清空</Button></div>
  <UiSelect v-model="source"><SelectTrigger aria-label="入口来源"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="all">全部来源</SelectItem><SelectItem value="private">私人入口</SelectItem><SelectItem value="public">公司收藏</SelectItem></SelectContent></UiSelect>
  <div class="relative"><Input v-model="group" class="pr-16" aria-label="筛选入口分组" placeholder="输入分组名称筛选" @compositionstart="composing=true" @compositionend="composing=false" /><Button v-if="group" variant="ghost" size="sm" class="absolute right-0 top-0" aria-label="清空分组筛选" @click="clearFilter('group')">清空</Button></div>
 </div>
 <p v-if="trash" class="mb-4 text-sm text-muted-foreground">私人入口及随入口删除的服务器账号保留 7 天；此前单独删除的账号不会随入口恢复。本地账号独立保留，可在“我的账号”中查看。</p>
 <div v-if="error" role="alert" class="notice notice-error">{{error}}<Button variant="ghost" @click="load">重试</Button></div>
 <p class="mb-4 min-h-5 text-sm text-muted-foreground" role="status">{{loading?'正在加载入口…':`共 ${total} 个入口`}}</p>
 <EmptyState v-if="!loading&&!error&&!rows.length" :icon="Link2" :title="trash?'回收站为空':keyword||group||source!=='all'?'没有匹配的入口':'还没有入口'" description="添加自己的网址，或在项目、文档和工具中点击收藏。" />
 <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
  <article v-for="entry in rows" :key="entry.id" class="min-w-0 rounded-xl border bg-card p-6" :aria-label="entry.name">
   <div class="mb-3 flex items-center justify-between gap-3 text-xs text-muted-foreground"><span>{{label(entry)}}{{entry.groupName?` · ${entry.groupName}`:''}}</span><Pin v-if="entry.pinned" class="size-4" aria-label="已置顶" /></div>
   <div class="mb-3 flex items-center gap-3"><ProjectIdentity :id="`${entry.source}:${entry.id}`" :name="entry.name" small /><h2 class="break-words text-base font-semibold">{{entry.name}}</h2></div>
   <p class="mb-3 break-all text-sm text-muted-foreground">{{entry.url||'原入口已不可用'}}</p>
   <p v-if="entry.note" class="mb-4 whitespace-pre-wrap break-words text-sm">{{entry.note}}</p>
   <p v-if="!entry.available&&!trash" class="mb-3 text-sm text-muted-foreground">原内容已删除或停用，可保留记录或取消收藏。</p>
   <div class="mt-4 flex flex-wrap gap-1">
    <template v-if="!trash"><Button variant="outline" size="sm" :disabled="!entry.available" @click="open(entry)"><ExternalLink class="size-4" />打开</Button><Button variant="ghost" size="sm" :disabled="!entry.available" @click="copy(entry)"><Copy class="size-4" />复制</Button>
     <RouterLink v-if="entry.available&&entry.source!=='document'" :class="buttonVariants({variant:'ghost',size:'sm'})" :to="accountPath(entry)"><KeyRound class="size-4" />我的账号</RouterLink>
     <Button variant="ghost" size="sm" :disabled="busy" @click="pin(entry)">{{entry.pinned?'取消置顶':'置顶'}}</Button><Button variant="ghost" size="sm" @click="edit(entry)"><PencilLine class="size-4" />编辑</Button>
     <Button v-if="entry.source==='private'" variant="ghost" size="sm" @click="deleting=entry;deleteError=''"><Trash2 class="size-4" />删除</Button><Button v-else variant="ghost" size="sm" :disabled="busy" @click="unfavorite(entry)">取消收藏</Button>
    </template><Button v-else variant="outline" :disabled="busy" @click="restore(entry)">恢复入口和账号</Button>
   </div>
  </article>
 </div>
 <div v-if="total>24" class="pagination"><span>第 {{page}} / {{Math.ceil(total/24)}} 页</span><Button variant="outline" :disabled="loading||page<=1" @click="page--">上一页</Button><Button variant="outline" :disabled="loading||page*24>=total" @click="page++">下一页</Button></div>
 <EntryFormDialog v-model:open="editorOpen" :entry="editing" @saved="saved" />
 <Dialog :open="!!deleting" @update:open="!$event&&!busy&&(deleting=null)"><DialogContent><DialogHeader><DialogTitle>删除私人入口</DialogTitle><DialogDescription>将“{{deleting?.name}}”及其下当前有效的服务器账号一起移入回收站，7 天内可一起恢复。本地账号独立保留，可在“我的账号”中查看。</DialogDescription></DialogHeader><p v-if="deleteError" role="alert" class="notice notice-error">{{deleteError}}</p><DialogFooter><Button variant="outline" :disabled="busy" @click="deleting=null">取消</Button><Button variant="destructive" :disabled="busy" @click="remove">移入回收站</Button></DialogFooter></DialogContent></Dialog>
</template>
