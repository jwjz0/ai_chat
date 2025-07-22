<template>
  <div class="block block-1" ref="assistantListContainer">
    <div class="list-header">
      <h3 class="list-title">{{ assistants.length ? '助手列表' : '暂无助手' }}</h3>
      <div class="action-buttons">
        <button class="add-btn" @click="$emit('add')">
          ➕ 新增助手
        </button>
        <button class="refresh-btn" @click="$emit('refresh')">
          ↺ 刷新
        </button>
      </div>
    </div>
    
    <div class="assistants-scroll-container">
      <div class="assistants-container">
        <div 
          v-for="assistant in assistants" 
          :key="assistant.id" 
          class="assistant-item"
          :class="{ active: selectedId === assistant.id }"
        >
          <div 
            class="assistant-content"
            @click="$emit('select', assistant)"
          >
            <div class="assistant-info">
              <p class="assistant-name">{{ assistant.name }}</p>
              <p class="assistant-desc">{{ assistant.description }}</p>
            </div>
            <div class="assistant-meta">
              <span>最新互动: {{ assistant.time_stamp }}</span>
            </div>
          </div>
          
          <div class="assistant-actions">
            <button 
              class="action-btn edit-btn"
              @click.stop="$emit('edit', assistant)"
              title="编辑"
            >
              ✏️
            </button>
            <button 
              class="action-btn delete-btn"
              @click.stop="$emit('delete', assistant.id)"
              title="删除"
            >
              🗑️
            </button>
          </div>
        </div>
      </div>
      
      <div v-if="loading" class="status loading">加载中...</div>
      <div v-else-if="assistants.length === 0" class="status empty">暂无助手数据</div>
      <div v-else-if="error" class="status error">{{ error }}</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, nextTick } from 'vue';

const props = defineProps({
  assistants: { type: Array, default: () => [] },
  selectedId: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  error: { type: String, default: '' }
});

const assistantListContainer = ref(null);
const scrollContainer = ref(null);
const isInitialLoad = ref(true);
const isUserInteraction = ref(false);

// 监听选中助手变化
watch(() => props.selectedId, (newVal, oldVal) => {
  if (oldVal) {
    isUserInteraction.value = true;
  }
  scrollToSelectedAssistant();
});

// 监听助手列表变化
watch(() => props.assistants, (newAssistants) => {
  if (isInitialLoad.value && newAssistants.length > 0) {
    isInitialLoad.value = false;
    scrollToSelectedAssistant();
  }
}, { deep: true });

const scrollToSelectedAssistant = async () => {
  await nextTick();
  
  if (!assistantListContainer.value) return;
  
  const selectedElement = assistantListContainer.value.querySelector('.assistant-item.active');
  if (!selectedElement) return;
  
  const container = scrollContainer.value || assistantListContainer.value.querySelector('.assistants-scroll-container');
  if (!container) return;
  
  const containerRect = container.getBoundingClientRect();
  const elementRect = selectedElement.getBoundingClientRect();
  
  const containerHeight = containerRect.height;
  const elementHeight = elementRect.height;
  const scrollTop = container.scrollTop;
  const targetScrollTop = scrollTop + elementRect.top - containerRect.top - (containerHeight - elementHeight) / 2;
  
  container.scrollTo({
    top: targetScrollTop,
    behavior: isInitialLoad.value || !isUserInteraction.value ? 'auto' : 'smooth'
  });
  
  if (isUserInteraction.value) {
    setTimeout(() => {
      isUserInteraction.value = false;
    }, 500);
  }
};

onMounted(() => {
  nextTick(() => {
    scrollToSelectedAssistant();
  });
});
</script>

<style scoped>
.block-1 {
  width: 280px;
  height: 100%;
  background-color: #1e293b; /* 深色背景稍浅一点，更柔和 */
  box-shadow: 2px 0 10px rgba(0, 0, 0, 0.1);
  z-index: 10;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.list-header {
  padding: 16px;
  background-color: #1e293b;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08); /* 边框更淡 */
  position: sticky;
  top: 0;
  z-index: 20;
}

.list-title {
  color: #f8fafc; /* 文字更亮一点 */
  margin: 0 0 12px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.action-buttons {
  display: flex;
  gap: 8px;
  margin-bottom: 0;
}

.add-btn {
  /* 薄荷绿主色 */
  background-color: #4ade80;
  border: none;
  color: #0f172a; /* 深色文字更搭配浅色按钮 */
  padding: 6px 10px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  flex: 1;
  justify-content: center;
  transition: all 0.3s;
  font-weight: 500;
}

.add-btn:hover {
  background-color: #22c55e; /* 深一点的绿色 */
  transform: translateY(-1px);
}

.refresh-btn {
  background-color: rgba(255, 255, 255, 0.1);
  border: none;
  color: #f8fafc;
  padding: 6px 10px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  transition: all 0.3s;
}

.refresh-btn:hover {
  background-color: rgba(255, 255, 255, 0.15);
  transform: translateY(-1px);
}

.assistants-scroll-container {
  flex: 1;
  overflow-y: auto;
  padding: 0 16px 16px;
}

.assistants-scroll-container::-webkit-scrollbar {
  display: block;
  width: 6px;
}

.assistants-scroll-container::-webkit-scrollbar-thumb {
  background-color: rgba(74, 222, 128, 0.3); /* 滚动条用主题色 */
  border-radius: 3px;
}

.assistants-container {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 16px;
  scroll-margin: 20px;
}

.assistant-item {
  background-color: rgba(255, 255, 255, 0.05); /* 项背景更淡 */
  border-radius: 6px;
  padding: 12px;
  color: #f8fafc;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
  overflow: hidden;
}

.assistant-item:hover {
  background-color: rgba(255, 255, 255, 0.08);
  transform: translateX(2px);
}

.assistant-item.active {
  /* 选中状态用主题色 */
  background-color: rgba(74, 222, 128, 0.15);
  border-left: 3px solid #4ade80;
}

.assistant-content {
  cursor: pointer;
  padding-right: 40px;
}

.assistant-info {
  margin-bottom: 8px;
}

.assistant-name {
  font-weight: 600;
  margin: 0 0 4px 0;
  font-size: 16px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 180px;
}

.assistant-desc {
  margin: 0;
  font-size: 14px;
  color: rgba(255, 255, 255, 0.7);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.assistant-meta {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
}

.assistant-actions {
  position: absolute;
  top: 12px;
  right: 12px;
  display: flex;
  gap: 6px;
  opacity: 0;
  transform: translateX(10px);
  transition: all 0.2s ease;
}

.assistant-item:hover .assistant-actions {
  opacity: 1;
  transform: translateX(0);
}

.action-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.edit-btn {
  background-color: rgba(255, 255, 255, 0.1);
  color: #f8fafc;
}

.edit-btn:hover {
  background-color: rgba(255, 255, 255, 0.15);
}

.delete-btn {
  background-color: rgba(239, 68, 68, 0.2); /* 红色调淡一点 */
  color: #f8fafc;
}

.delete-btn:hover {
  background-color: rgba(239, 68, 68, 0.3);
}

.status {
  text-align: center;
  padding: 20px 0;
  font-size: 14px;
}

/* 加载状态用主题色 */
.loading { color: #4ade80; }
.empty { color: rgba(255, 255, 255, 0.5); }
.error { color: #ef4444; }
</style>