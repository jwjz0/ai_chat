<template>
  <div class="container">
    <!-- 左侧助手列表 -->
    <div class="block block-1" ref="assistantListContainer">
      <!-- 固定头部 -->
      <div class="list-header">
        <h3 class="list-title">{{ sortedAssistants.length ? '助手列表' : '暂无助手' }}</h3>
        <div class="action-buttons">
          <button class="add-btn" @click="openAddModal">
            ➕ 新增助手
          </button>
          <button class="refresh-btn" @click="fetchAssistants">
            ↺ 刷新
          </button>
        </div>
      </div>
      
      <!-- 滚动内容区 -->
      <div class="assistants-scroll-container">
        <div class="assistants-container">
          <div 
            v-for="assistant in sortedAssistants" 
            :key="assistant.id" 
            class="assistant-item"
            :class="{ active: selectedAssistantId === assistant.id }"
            :ref="(el) => assistantRefs[assistant.id] = el"
          >
            <div 
              class="assistant-content"
              @click="handleSelectAssistant(assistant)"
            >
              <div class="assistant-info">
                <p class="assistant-name">{{ assistant.name }}</p>
                <p class="assistant-desc">{{ assistant.description }}</p>
              </div>
              <div class="assistant-meta">
                <span>最新互动: {{ formatTime(assistant.time_stamp) }}</span>
              </div>
            </div>
            
            <div class="assistant-actions">
              <button 
                class="action-btn edit-btn"
                @click.stop="handleEdit(assistant)"
                title="编辑"
              >
                ✏️
              </button>
              <button 
                class="action-btn delete-btn"
                @click.stop="handleDelete(assistant.id)"
                title="删除"
              >
                🗑️
              </button>
            </div>
          </div>
        </div>
        
        <div v-if="loading" class="status loading">加载中...</div>
        <div v-else-if="sortedAssistants.length === 0" class="status empty">暂无助手数据</div>
        <div v-else-if="error" class="status error">{{ error }}</div>
      </div>
    </div>

    <div class="right-container">
      <div class="content-wrapper">
        <!-- 历史对话区域 -->
        <div class="block block-2">
          <!-- 固定头部 -->
          <div class="history-header sticky-header">
            <h3>
              {{ selectedAssistant ? selectedAssistant.name + ' 的对话' : '历史对话' }}
            </h3>
            <div class="header-actions">
              <button 
                class="reset-btn" 
                :disabled="!selectedAssistantId"
                @click="handleResetHistory"
              >
                重置对话
              </button>
            </div>
          </div>
          
          <!-- 滚动内容区 -->
          <div class="history-scroll-container">
            <div v-if="selectedAssistantId" class="history-stats">
              消息数: {{ historyData.messages?.length || 0 }} | 
              总tokens: {{ totalTokens }}
            </div>
            
            <div class="history-container" ref="historyContainer">
              <div v-if="!selectedAssistantId" class="empty-state">
                请从左侧选择一个助手
              </div>
              
              <div v-else-if="loadingHistory" class="loading-state">
                <div class="spinner"></div>
                <p>加载历史中...</p>
              </div>
              
              <div v-else-if="historyData.messages && historyData.messages.length">
                <div 
                  v-for="(msg, index) in historyData.messages" 
                  :key="index" 
                  class="message-item"
                  :id="'msg-' + index"
                >
                  <div class="message-time">{{ formatTime(msg.gmt_create) }}</div>
                  
                  <!-- 用户消息（右侧） -->
                  <div v-if="msg.input.send" class="user-message-container">
                    <div class="message-content-wrapper">
                      <div class="message-bubble user-bubble">
                        <div class="message-content">{{ msg.input.send }}</div>
                        <div class="message-meta" style="text-align: right;" v-if="msg.usage.input_tokens > 0">
                          输入tokens: {{ msg.usage.input_tokens }}
                        </div>
                      </div>
                      <!-- 用户头像 -->
                      <div class="user-avatar">
                        <div class="avatar-image">👤</div>
                      </div>
                    </div>
                  </div>
                  
                  <!-- 助手消息（左侧） -->
                  <div v-if="msg.output.content" class="assistant-message-container">
                    <div class="message-content-wrapper">
                      <!-- 助手头像 -->
                      <div class="assistant-avatar">
                        <div class="avatar-image">🤖</div>
                      </div>
                      <div class="message-bubble assistant-bubble">
                        <div class="message-content">
                          <template v-if="msg.isLoading">
                            {{ msg.output.content }}
                            <span class="typing-indicator">
                              <span>.</span><span>.</span><span>.</span>
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
              
              <div v-else-if="selectedAssistantId" class="empty-state">
                该助手暂无历史对话
              </div>
              
              <!-- 新消息指示器 -->
              <div 
                v-if="!autoScroll && historyData.messages.length > 0" 
                class="scroll-indicator"
                @click="scrollToBottom(true)"
              >
                <span>有新消息</span>
                <i class="arrow-down"></i>
              </div>
            </div>
          </div>
        </div>

        <!-- 输入区 -->
        <div class="block block-3">
          <div class="input-container">
            <textarea
              v-model="userInput"
              class="user-input"
              placeholder="请输入消息... (Enter发送, Shift+Enter换行)"
              @keydown.enter.prevent="handleKeydown"
              :disabled="!selectedAssistantId || sending"
            ></textarea>
            <button 
              class="send-btn" 
              @click="sendMessage"
              :disabled="!selectedAssistantId || !userInput.trim() || sending"
              :title="sending ? '发送中...' : '发送消息'"
            >
              <span v-if="sending" class="loading-spinner">↑</span>
              <span v-else>↑</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 模态框 -->
    <div 
      v-if="isModalOpen" 
      class="modal-overlay"
      @click="closeModal"
    >
      <div 
        class="modal-content"
        @click.stop
      >
        <div class="modal-header">
          <h3>{{ currentAssistant.id ? '编辑助手' : '新增助手' }}</h3>
          <button class="close-btn" @click="closeModal">×</button>
        </div>
        
        <div class="modal-body">
          <form @submit.prevent="saveAssistant">
            <div class="form-group">
              <label>名称 <span class="required">*</span></label>
              <input
                type="text"
                v-model="currentAssistant.name"
                required
                placeholder="请输入助手名称"
              >
            </div>
            
            <div class="form-group">
              <label>描述</label>
              <textarea
                v-model="currentAssistant.description"
                placeholder="请输入助手描述"
                rows="2"
              ></textarea>
            </div>
            
            <div class="form-group">
              <label>提示词 <span class="required">*</span></label>
              <textarea
                v-model="currentAssistant.prompt"
                required
                placeholder="请输入提示词"
                rows="5"
              ></textarea>
            </div>
            
            <div class="form-actions">
              <button type="button" class="cancel-btn" @click="closeModal">取消</button>
              <button type="submit" class="save-btn">保存</button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, reactive, computed, nextTick, watch, onUpdated } from 'vue'
import assistantApi from '@/api/assistant'
import historyApi from '@/api/history'

// 助手列表数据
const assistants = ref([])
const loading = ref(false)
const error = ref('')
const selectedAssistantId = ref('')
const assistantListContainer = ref(null)
const assistantRefs = ref({})

// 当前选中的助手（用于动态标题）
const selectedAssistant = computed(() => {
  return assistants.value.find(assist => assist.id === selectedAssistantId.value) || null
})

// 历史对话数据
const historyData = reactive({
  messages: []
})
const loadingHistory = ref(false)

// 输入区状态
const userInput = ref('')
const sending = ref(false)
const streamController = ref(null)

// 滚动控制
const historyContainer = ref(null)
const autoScroll = ref(true)
const isScrolling = ref(false)
let scrollTimeout = null

// 模态框状态
const isModalOpen = ref(false)
const currentAssistant = reactive({
  id: '',
  name: '',
  description: '',
  prompt: '',
  gmt_create: '',
  gmt_modified: '',
  time_stamp: ''
})

// 计算总tokens
const totalTokens = computed(() => {
  return historyData.messages?.reduce((sum, msg) => {
    return sum + (msg.usage?.total_tokens || 0)
  }, 0) || 0
})

// 排序后的助手列表
const sortedAssistants = computed(() => {
  return [...assistants.value].sort((a, b) => {
    return new Date(b.time_stamp) - new Date(a.time_stamp)
  })
})

// 处理键盘事件 (Enter发送, Shift+Enter换行)
const handleKeydown = (e) => {
  if (e.shiftKey) {
    // Shift+Enter 换行
    const cursorPos = e.target.selectionStart;
    const textBefore = userInput.value.substring(0, cursorPos);
    const textAfter = userInput.value.substring(cursorPos);
    userInput.value = textBefore + '\n' + textAfter;
    // 移动光标到换行后
    nextTick(() => {
      e.target.selectionStart = e.target.selectionEnd = cursorPos + 1;
    });
  } else {
    // 单独Enter 发送消息
    sendMessage();
  }
}

// 获取助手列表
const fetchAssistants = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await assistantApi.getAll()
    assistants.value = res.data || res
    if (assistants.value.length > 0 && !selectedAssistantId.value) {
      const latestAssistant = assistants.value.reduce((latest, curr) => {
        return new Date(curr.time_stamp) > new Date(latest.time_stamp) 
          ? curr 
          : latest
      }, assistants.value[0])
      handleSelectAssistant(latestAssistant)
    }
  } catch (err) {
    error.value = err.message || '获取助手列表失败'
    console.error('获取助手失败:', err)
  } finally {
    loading.value = false
  }
}

// 选择助手
const handleSelectAssistant = async (assistant) => {
  selectedAssistantId.value = assistant.id
  historyData.messages = []
  loadingHistory.value = true
  
  try {
    const res = await historyApi.getByAssistantId(assistant.id)
    const newHistory = res.data || res
    if (selectedAssistantId.value === assistant.id) {
      historyData.messages = newHistory.messages || []
      scrollToAssistant(assistant.id)
      await nextTick()
      scrollToBottom(true)
    }
  } catch (err) {
    console.error('获取历史失败:', err)
  } finally {
    if (selectedAssistantId.value === assistant.id) {
      loadingHistory.value = false
    }
  }
}

// 编辑助手
const handleEdit = (assistant) => {
  Object.assign(currentAssistant, { ...assistant })
  isModalOpen.value = true
}

// 删除助手
const handleDelete = async (id) => {
  if (!confirm('确定要删除该助手吗？删除后对话记录将一并清除！')) return
  
  try {
    await assistantApi.deleteById(id)
    if (id === selectedAssistantId.value) {
      selectedAssistantId.value = ''
      historyData.messages = []
    }
    fetchAssistants()
  } catch (err) {
    alert('删除失败: ' + (err.message || '未知错误'))
    console.error('删除助手失败:', err)
  }
}

// 打开新增模态框
const openAddModal = () => {
  const now = new Date().toISOString().replace('T', ' ')
  Object.assign(currentAssistant, {
    id: '',
    name: '',
    description: '',
    prompt: '',
    gmt_create: now,
    gmt_modified: now,
    time_stamp: now
  })
  isModalOpen.value = true
}

// 关闭模态框
const closeModal = () => {
  isModalOpen.value = false
}

// 保存助手
const saveAssistant = async () => {
  try {
    const now = new Date().toISOString().replace('T', ' ')
    const payload = { ...currentAssistant }
    payload.gmt_modified = now
    payload.time_stamp = now

    if (currentAssistant.id) {
      await assistantApi.updateById(currentAssistant.id, payload)
    } else {
      payload.gmt_create = now
      await assistantApi.save(payload)
    }
    
    await fetchAssistants()
    const targetId = currentAssistant.id || sortedAssistants.value[0]?.id
    if (targetId) {
      scrollToAssistant(targetId)
    }
    closeModal()
  } catch (err) {
    alert('保存失败: ' + (err.message || '未知错误'))
    console.error('保存助手失败:', err)
  }
}

// 重置对话
const handleResetHistory = async () => {
  if (!selectedAssistantId.value) return
  if (!confirm('确定要重置该助手的所有对话记录吗？')) return
  
  try {
    await historyApi.resetByAssistantId(selectedAssistantId.value)
    loadingHistory.value = true
    const res = await historyApi.getByAssistantId(selectedAssistantId.value)
    historyData.messages = res.data?.messages || res.messages || []
    
    const now = new Date().toISOString().replace('T', ' ')
    assistants.value = assistants.value.map(assist => 
      assist.id === selectedAssistantId.value 
        ? { ...assist, time_stamp: now } 
        : assist
    )
    
    await nextTick()
    scrollToAssistant(selectedAssistantId.value)
    scrollToBottom(true)
  } catch (err) {
    alert('重置失败: ' + (err.message || '未知错误'))
  } finally {
    if (selectedAssistantId.value) {
      loadingHistory.value = false
    }
  }
}

// 发送消息
const sendMessage = async () => {
  if (!selectedAssistantId.value) {
    alert('请先从左侧选择一个助手');
    return;
  }
  
  const inputText = userInput.value.trim();
  if (!inputText) return;
  
  sending.value = true;
  
  const currentAssist = assistants.value.find(a => a.id === selectedAssistantId.value);
  if (!currentAssist) {
    alert('未找到当前助手信息');
    sending.value = false;
    return;
  }  
  // 添加用户消息
  const userMessage = {
    input: { prompt: currentAssist.prompt, send: inputText },
    output: { content: '' },
    usage: { input_tokens: 0, output_tokens: 0, total_tokens: 0 },
  };
  
  historyData.messages.push(userMessage);
  await scrollToBottom(true);
  
  // 添加助手"加载中"消息
  const loadingMessage = {
    input: { prompt: '', send: '' },
    output: { content: '' },
    usage: { input_tokens: 0, output_tokens: 0, total_tokens: 0 },
    isLoading: true
  };
  
  historyData.messages.push(loadingMessage);
  await scrollToBottom(true);

  let fullContent = '';
  let isDone = false;

  // 调用流式接口
  streamController.value = historyApi.streamProcessMessage(
    selectedAssistantId.value,
    { prompt: currentAssist.prompt, send: inputText },
    async (content) => {
      fullContent += content;
      
      historyData.messages[historyData.messages.length - 1] = {
        ...loadingMessage,
        output: { content: fullContent },
        isLoading: true
      };
      
      await nextTick();
      
      if (autoScroll.value) {
        scrollToBottom();
      }
    },
    (error) => {
      console.error('流式请求错误:', error);
      alert('发送失败: ' + error);
      historyData.messages.pop();
      sending.value = false;
    },
    async (usage) => {
      isDone = true;
      
      // 计算token
      const userInputTokens = Math.ceil(inputText.length / 4);
      const assistantOutputTokens = Math.ceil(fullContent.length / 4);
      const totalTokens = userInputTokens + assistantOutputTokens;

      // 更新消息token
      historyData.messages[historyData.messages.length - 2].usage = {
        input_tokens: userInputTokens,
        output_tokens: 0,
        total_tokens: totalTokens
      };

      // 更新助手消息
      historyData.messages[historyData.messages.length - 1] = {
        input: { prompt: currentAssist.prompt, send: '' },
        output: { content: fullContent },
        usage: {
          input_tokens: userInputTokens,
          output_tokens: assistantOutputTokens,
          total_tokens: totalTokens
        },
        isLoading: false
      };

      // 强制刷新UI
      historyData.messages = [...historyData.messages];
      
      // 更新助手最新互动时间
      const nowUpdate = new Date().toISOString().replace('T', ' ');
      assistants.value = assistants.value.map(assist => 
        assist.id === selectedAssistantId.value 
          ? { ...assist, time_stamp: nowUpdate } 
          : assist
      );

      userInput.value = '';
      sending.value = false;
      
      // 确保最终结果可见
      await scrollToBottom(true);
    }
  );
}

// 取消当前流式请求
const cancelStream = () => {
  if (streamController.value) {
    streamController.value.abort()
    streamController.value = null
    sending.value = false
    historyData.messages.forEach(msg => {
      if (msg.isLoading) {
        msg.isLoading = false;
      }
    });
  }
}

// 滚动到指定助手
const scrollToAssistant = (assistantId) => {
  nextTick(() => {
    const assistantEl = assistantRefs.value[assistantId];
    if (assistantEl && assistantListContainer.value) {
      assistantEl.scrollIntoView({ 
        behavior: 'smooth',
        block: 'center'
      });
    }
  });
}

// 滚动到历史底部
const scrollToBottom = async (force = false) => {
  if (!force && historyContainer.value) {
    const container = historyContainer.value;
    const scrollBottom = container.scrollHeight - container.scrollTop;
    const isAtBottom = scrollBottom <= container.clientHeight + 50;
    
    if (!isAtBottom) {
      autoScroll.value = false;
      return;
    }
  }
  
  await nextTick();
  if (historyContainer.value) {
    const container = historyContainer.value;
    
    const lastMessage = document.getElementById(`msg-${historyData.messages.length - 1}`);
    if (lastMessage) {
      lastMessage.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    } else {
      container.scrollTop = container.scrollHeight;
    }
    
    autoScroll.value = true;
  }
}

// 时间格式化
const formatTime = (timeStr) => {
  if (!timeStr) return '';
  return timeStr.replace('T', ' ').slice(0, 19);
}

// 监听滚动事件
const setupScrollListener = () => {
  if (!historyContainer.value) return;
  
  historyContainer.value.addEventListener('scroll', () => {
    if (scrollTimeout) clearTimeout(scrollTimeout);
    
    isScrolling.value = true;
    
    scrollTimeout = setTimeout(() => {
      isScrolling.value = false;
      
      if (historyContainer.value) {
        const container = historyContainer.value;
        const scrollBottom = container.scrollHeight - container.scrollTop;
        const isAtBottom = scrollBottom <= container.clientHeight + 50;
        
        if (isAtBottom) {
          autoScroll.value = true;
        }
      }
    }, 300);
  });
}

// 初始化
onMounted(() => {
  fetchAssistants();
  setupScrollListener();
  
  nextTick(() => {
    scrollToBottom(true);
  });
});

// 组件卸载时取消请求
onUnmounted(() => {
  cancelStream()
})

// 监听DOM更新
onUpdated(() => {
  if (autoScroll.value && !isScrolling.value) {
    scrollToBottom();
  }
})

// 监听消息变化
watch(() => historyData.messages.length, async () => {
  if (autoScroll.value && !sending.value) {
    await scrollToBottom();
  }
})
</script>

<style scoped>
/* 全局样式设置 */
* {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol";
  box-sizing: border-box;
}

/* 整体布局 */
.container {
  display: flex;
  width: 100vw;
  height: 100vh;
  margin: 0;
  padding: 0;
  background-color: #f3f4f6;
  overflow: hidden;
}

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

/* 助手列表固定头部 */
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

/* 助手列表滚动内容区 */
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

/* 右侧容器 */
.right-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* 内容包装器 - 统一对话区和输入区宽度 */
.content-wrapper {
  width: 100%;
  max-width: 900px;
  margin: 0 auto;
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 0 20px;
  box-sizing: border-box;
}

/* 历史对话区域（方块2）样式 */
.block-2 {
  flex: 1;
  background-color: transparent;
  box-sizing: border-box;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* 历史对话固定头部 */
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

/* 历史对话滚动内容区 */
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

/* 历史对话滚动条 - 一直隐藏 */
.history-scroll-container {
  -ms-overflow-style: none;  /* IE和Edge */
  scrollbar-width: none;  /* Firefox */
}

.history-scroll-container::-webkit-scrollbar {
  display: none;  /* 所有浏览器隐藏 */
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

@keyframes spin {
  to { transform: rotate(360deg); }
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

/* 用户消息容器 */
.user-message-container {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
}

/* 助手消息容器 */
.assistant-message-container {
  display: flex;
  justify-content: flex-start;
  margin-bottom: 12px;
}

.message-content-wrapper {
  display: flex;
  align-items: flex-start; /* 头像与气泡顶部对齐 */
  gap: 12px;
  max-width: 85%;
}

.message-bubble {
  padding: 12px 18px;
  margin: 4px 0;
  word-break: break-word;
  position: relative;
  flex: 1;
  /* 移除抖动动画，改为轻微阴影变化 */
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

/* 头像样式 */
.user-avatar, .assistant-avatar {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  margin-top: 4px; /* 微调位置，与气泡更协调 */
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

/* 打字机动画 */
.typing-indicator {
  display: inline-flex;
  gap: 4px;
  vertical-align: middle;
  margin-left: 4px;
}

.typing-indicator span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #94a3b8;
  animation: typing 1.4s infinite ease-in-out;
}

.typing-indicator span:nth-child(1) { animation-delay: 0s; }
.typing-indicator span:nth-child(2) { animation-delay: 0.2s; }
.typing-indicator span:nth-child(3) { animation-delay: 0.4s; }

@keyframes typing {
  0% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
  100% { transform: translateY(0); }
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 20px;
}

.modal-content {
  background-color: white;
  border-radius: 8px;
  width: 100%;
  max-width: 600px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.modal-header {
  padding: 16px 20px;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h3 {
  margin: 0;
  color: #334155;
  font-size: 18px;
}

.close-btn {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: #94a3b8;
  padding: 4px;
  line-height: 1;
}

.close-btn:hover {
  color: #334155;
}

.modal-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  color: #555;
  font-size: 14px;
}

.required {
  color: #ff4d4f;
}

.form-group input,
.form-group textarea {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  font-size: 14px;
  box-sizing: border-box;
  font-family: inherit;
}

.form-group textarea {
  resize: vertical;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
}

.cancel-btn {
  padding: 8px 16px;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  background-color: white;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.cancel-btn:hover {
  background-color: #f8fafc;
}

.save-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  background-color: #3498db;
  color: white;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.save-btn:hover {
  background-color: #2980b9;
}

/* 滚动指示器 */
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

/* 发送按钮箭头动画 */
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>