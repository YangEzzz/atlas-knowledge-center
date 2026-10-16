import { ref } from "vue";
export const searchOpen = ref(false);
export const searchKeyword = ref("");
export function openSearch(keyword = "") {
  searchKeyword.value = keyword;
  searchOpen.value = true;
}
