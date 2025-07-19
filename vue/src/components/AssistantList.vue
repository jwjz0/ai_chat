<template>
  <div class="block block-1" ref="assistantListContainer">
    <!-- 固定头部 -->
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
    
    <!-- 滚动内容区 -->
    <div class="assistants-scroll-container">
      <div class="assistants-container">
        <div 
          v-for="assistant in assistants" 
          :key="assistant.id" 
          class="assistant-item"
          :class="{ active: selectedId === assistant.id }"
          :ref="(el) => assistantRefs[assistant.id] = el"
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
import { ref } from 'vue';

const props = defineProps({
  assistants: {
    type: Array,
    default: () => []
  },
  selectedId: {
    type: String,
    default: ''
  },
  loading: {
    type: Boolean,
    default: false
  },
  error: {
    type: String,
    default: ''
  }
});

const assistantRefs = ref({});
const assistantListContainer = ref(null);
</script>

<style scoped>
/* 左侧助手列表（方块1）样式 */
.block-1 {
  width: 280px;
  height: 100%;
  background-color: #2c3e50;
  box-shadow: 2px 0 10px rgba(0, 0, 0, 0.1);
  z-index: 10;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 固定头部 */
.list-header {
  padding: 16px;
  background-color: #2c3e50;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  position: sticky;
  top: 0;
  z-index: 20;
}

.list-title {
  color: #ecf0f1;
  margin: 0 0 12px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.action-buttons {
  display: flex;
  gap: 8px;
  margin-bottom: 0;
}

.add-btn {
  background-color: #3498db;
  border: none;
  color: white;
  padding: 6px 10px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  flex: 1;
  justify-content: center;
  transition: background-color 0.3s;
}

.add-btn:hover {
  background-color: #2980b9;
}

.refresh-btn {
  background-color: rgba(255, 255, 255, 0.15);
  border: none;
  color: white;
  padding: 6px 10px;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  transition: background-color 0.3s;
}

.refresh-btn:hover {
  background-color: rgba(255, 255, 255, 0.25);
}

/* 滚动内容区 */
.assistants-scroll-container {
  flex: 1;
  overflow-y: auto;
  padding: 0 16px 16px;
}

.assistants-scroll-container {
  -ms-overflow-style: auto;  /* IE和Edge */
}

.assistants-scroll-container::-webkit-scrollbar {
  display: block;  /* Chrome, Safari和Opera 显示 */
  width: 6px;
}

.assistants-scroll-container::-webkit-scrollbar-thumb {
  background-color: rgba(156, 156, 156, 0.3);
  border-radius: 3px;
}

.assistants-container {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 16px;
}

.assistant-item {
  background-color: rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  padding: 12px;
  color: #ecf0f1;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}

.assistant-item:hover {
  background-color: rgba(255, 255, 255, 0.15);
  transform: translateX(3px);
}

.assistant-item.active {
  background-color: rgba(52, 152, 219, 0.2);
  border-left: 3px solid #3498db;
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
  background-color: rgba(255, 255, 255, 0.15);
  color: #ecf0f1;
}

.edit-btn:hover {
  background-color: rgba(255, 255, 255, 0.25);
}

.delete-btn {
  background-color: rgba(231, 76, 60, 0.2);
  color: #ecf0f1;
}

.delete-btn:hover {
  background-color: rgba(231, 76, 60, 0.3);
}

.status {
  text-align: center;
  padding: 20px 0;
  font-size: 14px;
}

.loading { color: #3498db; }
.empty { color: rgba(255, 255, 255, 0.5); }
.error { color: #e74c3c; }
</style>