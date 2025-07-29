<template>
  <div class="block block-1" ref="assistantListContainer">
    <div class="list-header">
      <h3 class="list-title">{{ assistants.length ? '助手列表' : '暂无助手' }}</h3>
      <div class="action-buttons">
        <button class="add-btn" @click="$emit('add')">
          新增助手
        </button>
        <button class="refresh-btn" @click="$emit('refresh')">
          刷新
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
          @click="$emit('select', assistant)"
        >
          <div class="assistant-content">
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

// 监听选中助手变化
watch(() => props.selectedId, (newVal) => {
  scrollToSelectedAssistant();
});

// 监听助手列表变化
watch(() => props.assistants, () => {
  scrollToSelectedAssistant();
}, { deep: true });

// 平滑滚动到选中的助手项
const scrollToSelectedAssistant = async () => {
  await nextTick();
  
  const container = scrollContainer.value || assistantListContainer.value?.querySelector('.assistants-scroll-container');
  const selectedElement = assistantListContainer.value?.querySelector('.assistant-item.active');
  
  if (!container || !selectedElement) return;
  
  // 计算目标滚动位置，使选中项居中显示
  const containerRect = container.getBoundingClientRect();
  const elementRect = selectedElement.getBoundingClientRect();
  
  const containerHeight = containerRect.height;
  const elementHeight = elementRect.height;
  const scrollTop = container.scrollTop;
  const targetScrollTop = scrollTop + elementRect.top - containerRect.top - (containerHeight - elementHeight) / 2;
  
  // 使用平滑滚动
  container.scrollTo({
    top: targetScrollTop,
    behavior: 'smooth'
  });
};

onMounted(() => {
  scrollToSelectedAssistant();
});
</script>

<style scoped>
.block-1 {
  width: 280px;
  height: 100%;
  background-color: #ffffff;
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.05);
  z-index: 10;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid #e5e7eb;
}

.list-header {
  padding: 16px;
  background-color: #ffffff;
  border-bottom: 1px solid #e5e7eb;
  position: sticky;
  top: 0;
  z-index: 20;
}

.list-title {
  color: #4b5563;
  margin: 0 0 12px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid #e5e7eb;
  font-size: 16px;
  font-weight: 500;
}

.action-buttons {
  display: flex;
  gap: 8px;
  margin-bottom: 0;
}

.add-btn {
  background-color: #4b5563;
  border: none;
  color: #ffffff;
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
  background-color: #374151;
  transform: translateY(-1px);
}

.refresh-btn {
  background-color: #f3f4f6;
  border: none;
  color: #6b7280;
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
  background-color: #e5e7eb;
  transform: translateY(-1px);
}

.assistants-scroll-container {
  flex: 1;
  overflow-y: auto;
  padding: 0 16px 16px;
  scroll-behavior: smooth;
}

.assistants-scroll-container::-webkit-scrollbar {
  display: block;
  width: 6px;
}

.assistants-scroll-container::-webkit-scrollbar-thumb {
  background-color: #d1d5db;
  border-radius: 3px;
}

.assistants-scroll-container::-webkit-scrollbar-track {
  background-color: #f9fafb;
}

.assistants-container {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 16px;
  scroll-margin: 20px;
}

.assistant-item {
  background-color: #f9fafb;
  border-radius: 6px;
  padding: 12px;
  color: #4b5563;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: flex-start;
}

.assistant-item:hover {
  background-color: #f3f4f6;
  transform: translateX(2px);
}

.assistant-item.active {
  background-color: #e5e7eb;
  border-left: 3px solid #4b5563;
}

.assistant-content {
  flex: 1;
  padding-right: 40px;
}

.assistant-info {
  margin-bottom: 8px;
}

/* 恢复文字超出省略号功能 - 关键修复 */
.assistant-name {
  font-weight: 600;
  margin: 0 0 4px 0;
  font-size: 16px;
  white-space: nowrap;  /* 禁止换行 */
  overflow: hidden;    /* 隐藏超出部分 */
  text-overflow: ellipsis;  /* 显示省略号 */
  max-width: 180px;    /* 限制最大宽度 */
}

.assistant-desc {
  margin: 0;
  font-size: 14px;
  color: #6b7280;
  white-space: nowrap;  /* 禁止换行 */
  overflow: hidden;    /* 隐藏超出部分 */
  text-overflow: ellipsis;  /* 显示省略号 */
  max-width: 180px;    /* 限制最大宽度 */
}

.assistant-meta {
  font-size: 12px;
  color: #9ca3af;
  white-space: nowrap;  /* 禁止换行 */
  overflow: hidden;    /* 隐藏超出部分 */
  text-overflow: ellipsis;  /* 显示省略号 */
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
  background-color: #e5e7eb;
  color: #6b7280;
}

.edit-btn:hover {
  background-color: #d1d5db;
}

.delete-btn {
  background-color: #fee2e2;
  color: #ef4444;
}

.delete-btn:hover {
  background-color: #fecaca;
}

.status {
  text-align: center;
  padding: 20px 0;
  font-size: 14px;
}

.loading { color: #4b5563; }
.empty { color: #9ca3af; }
.error { color: #ef4444; }
</style>