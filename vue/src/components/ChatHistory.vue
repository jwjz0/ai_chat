<template>
  <div class="block block-2">
    <div class="history-header sticky-header">
      <div class="header-content">
        <h3>
          {{ assistant ? assistant.name + ' 的对话' : '历史对话' }}
        </h3>
      </div>
      <div class="header-actions">
        <button 
          class="reset-btn" 
          :disabled="!assistant"
          @click="$emit('reset-history')"
        >
          <span>重置对话</span>
        </button>
      </div>
    </div>
    
    <div class="history-scroll-container">
      <div v-if="assistant" class="history-stats">
        消息数: {{ messages?.length || 0 }} | 
        总tokens: {{ totalTokens }}
      </div>
      
      <div class="history-container" ref="historyContainer">
        <div v-if="!assistant" class="empty-state">
          请从左侧选择一个助手
        </div>
        
        <div v-else-if="loading" class="loading-state">
          <div class="spinner"></div>
          <p>加载历史中...</p>
        </div>
        
        <div v-else-if="messages && messages.length">
          <div 
            v-for="msg in messages" 
            :key="msg.id" 
            class="message-item"
          >
            <div v-if="msg.gmt_create" class="message-time">{{ msg.gmt_create }}</div>
            
            <div v-if="msg.input.send" class="user-message-container">
              <div class="message-content-wrapper">
                <div class="message-bubble user-bubble">
                  <div class="message-content" v-html="formatMessage(msg.input.send)"></div>
                  <div class="message-meta" style="text-align: right;" v-if="msg.usage.input_tokens > 0">
                    输入tokens: {{ msg.usage.input_tokens }}
                  </div>
                </div>
                <!-- 用户头像替换为图片 -->
                <div class="user-avatar">
                  <div class="avatar-image">
                    <img 
                      src="/src/assets/imgs/user.jpg" 
                      alt="用户头像" 
                      class="avatar-img"
                    >
                  </div>
                </div>
              </div>
            </div>
            
            <div v-if="msg.output.content || msg.isLoading" class="assistant-message-container">
              <div class="message-content-wrapper">
                <!-- 助手头像替换为图片 -->
                <div class="assistant-avatar">
                  <div class="avatar-image">
                    <img 
                      src="/src/assets/imgs/assistant.jpg" 
                      alt="助手头像" 
                      class="avatar-img"
                    >
                  </div>
                </div>
                <div class="message-bubble assistant-bubble">
                  <div class="message-content">
                    <template v-if="msg.isLoading">
                      <span v-html="formatMessage(msg.output.content)"></span>
                      <span class="typing-indicator">
                        <span></span>
                        <span></span>
                        <span></span>
                      </span>
                    </template>
                    <template v-else>
                      <span v-html="formatMessage(msg.output.content)"></span>
                    </template>
                  </div>
                  <div v-if="!msg.isLoading && msg.usage.output_tokens > 0" class="message-meta">
                    输出tokens: {{ msg.usage.output_tokens }} | 
                    总tokens: {{ msg.usage.total_tokens }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        
        <div v-else-if="assistant" class="empty-state">
          该助手暂无历史对话
        </div>

        <div 
          v-if="!autoScroll && messages.length > 0" 
          class="scroll-indicator"
          @click="scrollToBottom(true)"
        >
          <span>有新消息</span>
          <i class="arrow-down"></i>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';

const props = defineProps({
  assistant: { type: Object, default: null },
  messages: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  totalTokens: { type: Number, default: 0 }
});

const historyContainer = ref(null);
const autoScroll = ref(true);

const formatMessage = (content) => {
  if (!content) return '';
  let formatted = content.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
  formatted = formatted.replace(/\n\n/g, '<br><br>').replace(/\n/g, '<br>');
  return formatted;
};

watch(() => props.messages.length, () => {
  if (historyContainer.value) {
    const container = historyContainer.value;
    const scrollBottom = container.scrollHeight - container.scrollTop;
    autoScroll.value = scrollBottom <= container.clientHeight + 50;
  }
});

const scrollToBottom = (force = false) => {
  if (historyContainer.value) {
    historyContainer.value.scrollTop = historyContainer.value.scrollHeight;
    autoScroll.value = true;
  }
};
</script>

<style scoped>
/* 基础布局样式 */
.block-2 {
  flex: 1;
  background-color: #f9fafb;
  box-sizing: border-box;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.sticky-header {
  position: sticky;
  top: 0;
  background-color: #ffffff;
  z-index: 20;
  padding: 12px 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border-radius: 0 0 12px 12px;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.header-content h3 {
  margin: 0;
  color: #4b5563;
  font-size: 16px;
  font-weight: 500;
  letter-spacing: 0.3px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.reset-btn {
  background-color: #f3f4f6;
  color: #6b7280;
  border: none;
  border-radius: 8px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 13px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;
}

.reset-btn:hover:not(:disabled) {
  background-color: #e5e7eb;
  color: #4b5563;
}

.reset-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.history-scroll-container {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  margin-top: 4px;
}

.history-stats {
  color: #6b7280;
  font-size: 13px;
  margin-bottom: 16px;
  padding: 0 16px;
}

.history-container {
  min-height: 200px;
  position: relative;
}

.history-scroll-container {
  -ms-overflow-style: none;
  scrollbar-width: none;
}

.history-scroll-container::-webkit-scrollbar {
  display: none;
}

.loading-state {
  text-align: center;
  padding: 60px 0;
  color: #6b7280;
}

.spinner {
  width: 24px;
  height: 24px;
  margin: 0 auto 12px;
  border: 3px solid rgba(209, 213, 219, 0.5);
  border-radius: 50%;
  border-top-color: #6b7280;
  animation: spin 1s linear infinite;
}

.message-item {
  margin-bottom: 16px;
  position: relative;
  padding: 0 16px;
}

.message-time {
  text-align: center;
  font-size: 12px;
  color: #9ca3af;
  margin-bottom: 8px;
  font-weight: 500;
}

.user-message-container {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 8px;
}

.assistant-message-container {
  display: flex;
  justify-content: flex-start;
  margin-bottom: 8px;
}

.message-content-wrapper {
  display: inline-flex;
  align-items: flex-start;
  gap: 10px;
  max-width: 85%;
}

.message-bubble {
  padding: 8px 14px;
  word-break: break-word;
  position: relative;
  transition: box-shadow 0.2s ease;
  line-height: 1.5;
  border-radius: 16px;
}

.message-bubble:hover {
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.user-bubble {
  background-color: #4b5563;
  color: #ffffff;
  border-radius: 16px 4px 16px 16px;
}

.assistant-bubble {
  background-color: #ffffff;
  color: #374151;
  border: 1px solid #e5e7eb;
  border-radius: 4px 16px 16px 16px;
}

.message-content {
  line-height: 1.5;
  margin-bottom: 4px;
  font-size: 15px;
}

.message-meta {
  font-size: 11px;
}

.user-bubble .message-meta {
  color: rgba(255, 255, 255, 0.8);
}

.assistant-bubble .message-meta {
  color: #9ca3af;
}

.empty-state {
  color: #6b7280;
  text-align: center;
  padding: 60px 0;
  font-size: 16px;
  background-color: #ffffff;
  border: 1px dashed #e5e7eb;
  border-radius: 12px;
  margin: 20px 16px;
}

/* 头像样式调整 - 适配图片 */
.user-avatar, .assistant-avatar {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  margin-top: 2px;
}

.avatar-image {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden; /* 确保图片不会超出圆形范围 */
  background-color: #e2e8f0; /* 图片加载前的占位背景 */
}

/* 头像图片样式 */
.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover; /* 保持图片比例并填满容器 */
  border-radius: 50%; /* 确保图片是圆形 */
}

.typing-indicator {
  display: inline-flex;
  gap: 4px;
  vertical-align: middle;
  margin-left: 4px;
}

.typing-indicator span {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background-color: #9ca3af;
  animation: wave 1.4s infinite ease-in-out;
}

.typing-indicator span:nth-child(2) { animation-delay: 0.2s; }
.typing-indicator span:nth-child(3) { animation-delay: 0.4s; }

.scroll-indicator {
  position: absolute;
  bottom: 20px;
  right: 20px;
  background-color: #4b5563;
  color: white;
  padding: 6px 12px;
  border-radius: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  z-index: 10;
  opacity: 0;
  transform: translateY(20px);
  transition: all 0.3s ease;
  font-size: 13px;
}

.scroll-indicator:hover {
  background-color: #374151;
}

.scroll-indicator.visible {
  opacity: 1;
  transform: translateY(0);
}

.arrow-down {
  width: 0; 
  height: 0; 
  border-left: 5px solid transparent;
  border-right: 5px solid transparent;
  border-top: 5px solid white;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

@keyframes wave {
  0%, 60%, 100% { transform: translateY(0); }
  30% { transform: translateY(-5px); }
}

.message-content strong {
  font-weight: 600;
  color: inherit;
}
</style>