<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from "vue";
import { useRoute, useRouter } from "vue-router";
import { getPublicLink } from "@/api/public-links";
import { getProject } from "@/api/projects";
import { getEntry } from "@/api/entries";
import { Button, buttonVariants } from "@/components/ui/button";
import PersonalAccounts from "@/components/project/PersonalAccounts.vue";
const route=useRoute(),router=useRouter();
const parse=(value:unknown)=>{const id=Number(value);return Number.isSafeInteger(id)&&id>0?id:undefined};
const toolId=computed(()=>parse(route.query.toolId)),projectId=computed(()=>parse(route.query.projectId)),entryId=computed(()=>parse(route.query.entryId));
const scoped=computed(()=>route.query.toolId!==undefined||route.query.projectId!==undefined||route.query.entryId!==undefined);
const name=ref(''),error=ref(''),loading=ref(false);let epoch=0;
async function loadOwner(){
 const ticket=++epoch;name.value='';error.value='';loading.value=false;
 const ids=[route.query.toolId,route.query.projectId,route.query.entryId].filter(value=>value!==undefined);
 if(ids.length>1||(ids.length&&!parse(ids[0]))){error.value='请选择一个有效的项目、工具或私人入口。';return}
 if(!scoped.value)return;
 loading.value=true;
 try{
  const result=entryId.value?await getEntry(entryId.value):toolId.value?await getPublicLink(toolId.value):await getProject(projectId.value!);
  if(entryId.value&&'source' in result.data&&result.data.source!=='private')throw new Error('请选择私人入口');
  if(ticket===epoch)name.value=result.data.name;
 }catch{if(ticket===epoch)error.value='关联内容无法加载或已删除，请重试。'}finally{if(ticket===epoch)loading.value=false}
}
watch([()=>route.query.toolId,()=>route.query.projectId,()=>route.query.entryId],loadOwner,{immediate:true});
onBeforeUnmount(()=>{epoch++});
function consumed(){void router.replace({query:{...route.query,create:undefined}})}
</script>
<template>
 <div>
  <div class="page-header"><div><h1>{{name?`${name} · 我的账号`:'我的账号'}}</h1><p>{{scoped?'仅展示你在此入口下的个人账号。':'集中管理项目、工具和私人入口中的个人账号。'}}</p></div><RouterLink v-if="scoped" :class="buttonVariants({variant:'outline'})" to="/my-accounts">全部账号</RouterLink></div>
  <p v-if="loading" role="status">正在加载关联内容…</p>
  <div v-else-if="error" role="alert" class="notice notice-error">{{error}}<Button variant="ghost" @click="loadOwner">重试</Button></div>
  <PersonalAccounts v-else :tool-id="toolId" :project-id="projectId" :entry-id="entryId" :auto-create="route.query.create==='1'" @create-consumed="consumed" />
 </div>
</template>
