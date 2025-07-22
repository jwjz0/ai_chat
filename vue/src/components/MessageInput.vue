<template>
  <div class="block block-3">
    <div class="input-container">
      <textarea
        ref="textareaRef"
        :value="input"
        class="user-input"
        placeholder="请输入消息... (Enter发送, Shift+Enter换行)"
        @input="handleInput"
        @keydown.enter="handleKeydown($event)"
        :disabled="disabled"
      ></textarea>
      <button 
        class="send-btn" 
        @click="$emit('send')"
        :disabled="!input.trim() || disabled"
        :title="sending ? '停止回答' : '发送消息'"
      >
        <span v-if="sending" class="loading-spinner">✖</span>
        <span v-else>↑</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, watch } from 'vue';

const props = defineProps({
  input: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  sending: { type: Boolean, default: false }
});

const emit = defineEmits(['input-change', 'send', 'keydown']);
const textareaRef = ref(null);

// 处理输入事件，动态调整高度
const handleInput = (e) => {
  emit('input-change', e.target.value);
  adjustTextareaHeight();
};

// 处理回车键事件
const handleKeydown = (e) => {
  e.preventDefault();
  
  if (e.shiftKey) {
    const cursorPos = e.target.selectionStart;
    const newInput = props.input.substring(0, cursorPos) + '\n' + props.input.substring(cursorPos);
    emit('input-change', newInput);
    // 调整光标位置
    nextTick(() => {
      e.target.selectionStart = e.target.selectionEnd = cursorPos + 1;
      adjustTextareaHeight();
    });
  } else {
    emit('send');
  }
};

// 动态调整文本框高度
const adjustTextareaHeight = () => {
  if (!textareaRef.value) return;
  
  // 重置高度获取正确滚动高度
  textareaRef.value.style.height = 'auto';
  const scrollHeight = textareaRef.value.scrollHeight;
  
  // 限制最小和最大高度
  const minHeight = 58;
  const maxHeight = 160;
  textareaRef.value.style.height = `${Math.min(Math.max(scrollHeight, minHeight), maxHeight)}px`;
};

// 监听输入值变化调整高度
watch(() => props.input, () => {
  nextTick(adjustTextareaHeight);
});
</script>

<style scoped>
/* 保持与原始样式一致 */
.block-3 {
  flex: none;
  background-color: transparent;
  padding: 0 0 20px 0;
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
  font-size: 16px;
  min-height: 58px;
  max-height: 160px;
  line-height: 1.6;
  box-sizing: border-box;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  background-color: white;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  
  /* 隐藏滚动条 */
  overflow: hidden;
}

.user-input::-webkit-scrollbar {
  display: none;
}

.user-input:focus {
  outline: none;
  border-color: #3498db;
  box-shadow: 0 0 0 3px rgba(52, 152, 219, 0.1);
}

.user-input::placeholder {
  color: #94a3b8;
  font-size: 15px;
  opacity: 0.8;
}

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
  
  line-height: 30px;
  padding-bottom: 4px;
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

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>