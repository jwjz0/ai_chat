<template>
  <div class="block block-2">
    <div class="history-header sticky-header">
      <h3>
        {{ assistant ? assistant.name + ' 的对话' : '历史对话' }}
      </h3>
      <div class="header-actions">
        <button 
          class="reset-btn" 
          :disabled="!assistant"
          @click="$emit('reset-history')"
        >
          重置对话
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
            :id="'msg-' + msg.id"
          >
            <!-- 只显示有gmt_create的消息时间 -->
            <div v-if="msg.gmt_create" class="message-time">{{ msg.gmt_create }}</div>
            
            <div v-if="msg.input.send" class="user-message-container">
              <div class="message-content-wrapper">
                <div class="message-bubble user-bubble">
                  <div class="message-content">{{ msg.input.send }}</div>
                  <div class="message-meta" style="text-align: right;" v-if="msg.usage.input_tokens > 0">
                    输入tokens: {{ msg.usage.input_tokens }}
                  </div>
                </div>
                <div class="user-avatar">
                  <div class="avatar-image">👤</div>
                </div>
              </div>
            </div>
            
            <div v-if="msg.output.content || msg.isLoading" class="assistant-message-container">
              <div class="message-content-wrapper">
                <div class="assistant-avatar">
                  <div class="avatar-image">🤖</div>
                </div>
                <div class="message-bubble assistant-bubble">
                  <div class="message-content">
                    <template v-if="msg.isLoading">
                      {{ msg.output.content }}
                      <span class="typing-indicator">
                        <span></span>
                        <span></span>
                        <span></span>
                      </span>
                    </template>
                    <template v-else>
                      {{ msg.output.content }}
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
import { ref, onMounted, watch } from 'vue';

const props = defineProps({
  assistant: { type: Object, default: null },
  messages: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  totalTokens: { type: Number, default: 0 }
});

const historyContainer = ref(null);
const autoScroll = ref(true);

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
/* 样式保持不变 */
.block-2 {
  flex: 1;
  background-color: transparent;
  box-sizing: border-box;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.sticky-header {
  position: sticky;
  top: 0;
  background-color: #f3f4f6;
  z-index: 20;
  padding: 20px 0 12px;
  margin-bottom: 0;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 8px;
  border-bottom: 1px solid #e2e8f0;
}

.history-header h3 {
  margin: 0;
  color: #334155;
  font-size: 18px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.reset-btn {
  background-color: #e74c3c;
  color: white;
  border: none;
  border-radius: 4px;
  padding: 4px 10px;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: background-color 0.2s;
}

.reset-btn:enabled:hover {
  background-color: #d43827;
}

.reset-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.history-scroll-container {
  flex: 1;
  overflow-y: auto;
  padding: 16px 0 20px;
}

.history-stats {
  color: #64748b;
  font-size: 14px;
  margin-bottom: 16px;
  padding: 4px 0;
}

.history-container {
  padding: 10px 0;
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
  color: #64748b;
}

.spinner {
  width: 24px;
  height: 24px;
  margin: 0 auto 12px;
  border: 3px solid rgba(52, 152, 219, 0.2);
  border-radius: 50%;
  border-top-color: #3498db;
  animation: spin 1s linear infinite;
}

.message-item {
  margin-bottom: 20px;
  position: relative;
}

.message-time {
  text-align: center;
  font-size: 12px;
  color: #94a3b8;
  margin-bottom: 8px;
  font-weight: 500;
}

.user-message-container {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

.assistant-message-container {
  display: flex;
  justify-content: flex-start;
  margin-bottom: 12px;
}

.message-content-wrapper {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  max-width: 85%;
}

.message-bubble {
  padding: 12px 18px;
  margin: 4px 0;
  word-break: break-word;
  position: relative;
  flex: 1;
  transition: box-shadow 0.2s ease;
}

.message-bubble:hover {
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.08);
}

.user-bubble {
  background-color: #3498db;
  color: white;
  border-radius: 18px 18px 4px 18px;
  box-shadow: 0 2px 4px rgba(52, 152, 219, 0.15);
}

.assistant-bubble {
  background-color: #ffffff;
  color: #334155;
  border: 1px solid #e2e8f0;
  border-radius: 18px 18px 18px 4px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.message-content {
  line-height: 1.6;
  margin-bottom: 6px;
  font-size: 15px;
}

.message-meta {
  font-size: 12px;
}

.user-bubble .message-meta {
  color: rgba(255, 255, 255, 0.8);
}

.assistant-bubble .message-meta {
  color: #94a3b8;
}

.empty-state {
  color: #64748b;
  text-align: center;
  padding: 60px 0;
  font-size: 16px;
  background-color: rgba(255, 255, 255, 0.5);
  border-radius: 12px;
  margin: 20px 0;
}

.user-avatar, .assistant-avatar {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  margin-top: 4px;
}

.avatar-image {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  background-color: #e2e8f0;
}

.user-avatar .avatar-image {
  background-color: #3498db;
  color: white;
}

.assistant-avatar .avatar-image {
  background-color: #2c3e50;
  color: white;
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
  background-color: #94a3b8;
  animation: wave 1.4s infinite ease-in-out;
}

.typing-indicator span:nth-child(2) { animation-delay: 0.2s; }
.typing-indicator span:nth-child(3) { animation-delay: 0.4s; }

.scroll-indicator {
  position: absolute;
  bottom: 80px;
  right: 20px;
  background-color: #3498db;
  color: white;
  padding: 8px 16px;
  border-radius: 20px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 10;
  opacity: 0;
  transform: translateY(20px);
  transition: all 0.3s ease;
}

.scroll-indicator:hover {
  background-color: #2980b9;
}

.scroll-indicator.visible {
  opacity: 1;
  transform: translateY(0);
}

.arrow-down {
  width: 0; 
  height: 0; 
  border-left: 6px solid transparent;
  border-right: 6px solid transparent;
  border-top: 6px solid white;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

@keyframes wave {
  0%, 60%, 100% { transform: translateY(0); }
  30% { transform: translateY(-5px); }
}
</style>