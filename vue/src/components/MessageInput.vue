<template>
  <div class="block block-3">
    <div class="input-container">
      <textarea
        :value="input"
        class="user-input"
        placeholder="请输入消息... (Enter发送, Shift+Enter换行)"
        @input="$emit('input-change', $event.target.value)"
        @keydown.enter.prevent="$emit('keydown', $event)"
        :disabled="disabled"
      ></textarea>
      <button 
        class="send-btn" 
        @click="$emit('send')"
        :disabled="!input.trim() || disabled"
        :title="sending ? '发送中...' : '发送消息'"
      >
        <span v-if="sending" class="loading-spinner">↑</span>
        <span v-else>↑</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { defineProps } from 'vue';

const props = defineProps({
  input: {
    type: String,
    default: ''
  },
  disabled: {
    type: Boolean,
    default: false
  },
  sending: {
    type: Boolean,
    default: false
  }
});
</script>

<style scoped>
/* 输入区（方块3）样式优化 */
.block-3 {
  flex: none;
  background-color: transparent;
  padding: 20px 0;
  box-sizing: border-box;
  display: flex;
  align-items: center;
}

.input-container {
  display: flex;
  gap: 12px;
  width: 100%;
  position: relative;
}

.user-input {
  flex: 1;
  padding: 15px 20px 15px 20px;
  border: 1px solid #d1d5db;
  border-radius: 28px;
  resize: none;
  font-size: 15px;
  min-height: 58px;
  max-height: 160px;
  line-height: 1.6;
  box-sizing: border-box;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  background-color: white;
}

.user-input:focus {
  outline: none;
  border-color: #3498db;
  box-shadow: 0 0 0 3px rgba(52, 152, 219, 0.1);
}

.user-input::placeholder {
  color: #94a3b8;
  font-size: 14px;
  opacity: 0.8;
}

/* 发送按钮 - 小、粗、实心箭头 */
.send-btn {
  position: absolute;
  bottom: 12px;
  right: 8px;
  width: 40px;
  height: 40px;
  background-color: #3498db;
  color: white;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  font-size: 25px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  box-shadow: 0 2px 5px rgba(52, 152, 219, 0.2);
  z-index: 10;
  font-weight: bold;
  
  /* 箭头位置调整 */
  line-height: 30px;  /* 减小行高使箭头整体上移 */
  padding-bottom: 4px;  /* 底部增加内边距 */
}

.send-btn:enabled:hover {
  background-color: #2980b9;
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(52, 152, 219, 0.25);
}

.send-btn:disabled {
  background-color: #cbd5e1;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

.loading-spinner {
  display: inline-block;
  animation: spin 1s linear infinite;
}

/* 发送按钮箭头动画 */
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>