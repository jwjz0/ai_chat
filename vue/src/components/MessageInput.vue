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
        @click="handleButtonClick"
        :disabled="(sending ? false : !input.trim()) || baseDisabled"
        :title="sending ? '停止回答' : '发送消息'"
      >
        <span v-if="sending" class="stop-icon">✖</span>
        <span v-else>↑</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, watch } from 'vue';

const props = defineProps({
  input: { type: String, default: '' },
  baseDisabled: { type: Boolean, default: false },
  sending: { type: Boolean, default: false }
});

const emit = defineEmits(['input-change', 'send', 'stop', 'keydown']);
const textareaRef = ref(null);

const handleButtonClick = () => {
  console.log('按钮点击，sending状态：', props.sending);
  if (props.sending) {
    emit('stop');
  } else {
    emit('send');
  }
};

const handleInput = (e) => {
  emit('input-change', e.target.value);
  adjustTextareaHeight();
};

const handleKeydown = (e) => {
  e.preventDefault();
  
  if (e.shiftKey) {
    const cursorPos = e.target.selectionStart;
    const newInput = props.input.substring(0, cursorPos) + '\n' + props.input.substring(cursorPos);
    emit('input-change', newInput);
    nextTick(() => {
      e.target.selectionStart = e.target.selectionEnd = cursorPos + 1;
      adjustTextareaHeight();
    });
  } else {
    emit('send');
  }
};

const adjustTextareaHeight = () => {
  if (!textareaRef.value) return;
  
  textareaRef.value.style.height = 'auto';
  const scrollHeight = textareaRef.value.scrollHeight;
  const minHeight = 58;
  const maxHeight = 160;
  textareaRef.value.style.height = `${Math.min(Math.max(scrollHeight, minHeight), maxHeight)}px`;
};

watch(() => props.input, () => {
  nextTick(adjustTextareaHeight);
});
</script>

<style scoped>
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
  font-size: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  box-shadow: 0 2px 5px rgba(52, 152, 219, 0.2);
  z-index: 10;
  font-weight: bold;
  line-height: 30px;
  padding-bottom: 0;
  cursor: pointer;
}

.send-btn:not(:disabled):hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(52, 152, 219, 0.25);
}

/* 移除了X按钮悬停时的红色圆形背景 */
.send-btn:disabled {
  background-color: #cbd5e1;
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}
</style>