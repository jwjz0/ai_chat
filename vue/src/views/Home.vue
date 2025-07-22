<template>
  <div class="container">
    <AssistantList 
      :assistants="sortedAssistants"
      :selected-id="selectedAssistantId"
      :loading="loading"
      :error="error"
      @select="handleSelectAssistant"
      @edit="handleEdit"
      @delete="handleDelete"
      @add="openAddModal"
      @refresh="fetchAssistants"
    />
    
    <div class="right-container">
      <div class="content-wrapper">
        <!-- 使用您提供的可正常显示历史的ChatHistory组件 -->
        <ChatHistory 
          :assistant="selectedAssistant"
          :messages="historyMessages"
          :loading="loadingHistory"
          :total-tokens="totalTokens"
          @reset-history="handleResetHistory"
        />
        
        <MessageInput 
          :input="userInput"
          :base-disabled="!selectedAssistantId"
          :sending="sending"
          @input-change="userInput = $event"
          @send="sendMessage"
          @stop="stopMessage"
          @keydown="handleKeydown"
        />
      </div>
    </div>
    
    <AssistantModal 
      :visible="isModalOpen"
      :assistant="currentAssistant"
      @close="closeModal"
      @save="saveAssistant"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, reactive, computed, nextTick, watch } from 'vue'
import assistantApi from '@/api/assistant'
import historyApi from '@/api/history'
import AssistantList from '@/components/AssistantList.vue'
import ChatHistory from '@/components/ChatHistory.vue' // 引入您提供的ChatHistory
import MessageInput from '@/components/MessageInput.vue'
import AssistantModal from '@/components/AssistantModal.vue'

// 状态管理
const assistants = ref([])
const loading = ref(false)
const error = ref('')
const selectedAssistantId = ref('')

const selectedAssistant = computed(() => {
  return assistants.value.find(assist => assist.id === selectedAssistantId.value) || null
})

// 历史消息数组（核心：保持与后端数据结构一致）
const historyMessages = ref([])
const loadingHistory = ref(false)

const userInput = ref('')
const sending = ref(false)
const streamAbortController = ref(null)

const autoScroll = ref(true)
const isScrolling = ref(false)
let scrollTimeout = null

const isModalOpen = ref(false)
const currentAssistant = reactive({
  id: '',
  name: '',
  description: '',
  prompt: '',
})

const totalTokens = computed(() => {
  return historyMessages.value?.reduce((sum, msg) => {
    return sum + (msg.usage?.total_tokens || 0)
  }, 0) || 0
})

const sortedAssistants = computed(() => {
  return [...assistants.value].sort((a, b) => {
    return new Date(b.time_stamp) - new Date(a.time_stamp)
  })
})

// 业务逻辑
const handleKeydown = (e) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

const fetchAssistants = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await assistantApi.getAll()
    assistants.value = res.data || res
    if (assistants.value.length > 0 && !selectedAssistantId.value) {
      const latestAssistant = assistants.value.reduce((latest, curr) => {
        return new Date(curr.time_stamp) > new Date(latest.time_stamp) ? curr : latest
      }, assistants.value[0])
      handleSelectAssistant(latestAssistant)
    }
  } catch (err) {
    error.value = err.message || '获取助手列表失败'
  } finally {
    loading.value = false
  }
}

// 修复：确保历史记录正确加载
const handleSelectAssistant = async (assistant) => {
  selectedAssistantId.value = assistant.id
  historyMessages.value = [] // 清空当前消息
  loadingHistory.value = true
  
  try {
    // 从后端获取历史消息（保持原始结构）
    const res = await historyApi.getByAssistantId(assistant.id)
    // 直接赋值后端返回的messages数组，不做额外修改
    historyMessages.value = res.data?.messages || res.messages || []
    await nextTick()
    scrollToBottom(true) // 滚动到底部
  } catch (err) {
    console.error('获取历史失败:', err)
  } finally {
    loadingHistory.value = false
  }
}

const handleEdit = (assistant) => {
  Object.assign(currentAssistant, { ...assistant })
  isModalOpen.value = true
}

const handleDelete = async (id) => {
  if (!confirm('确定要删除该助手吗？')) return
  
  try {
    await assistantApi.deleteById(id)
    if (id === selectedAssistantId.value) {
      selectedAssistantId.value = ''
      historyMessages.value = []
    }
    fetchAssistants()
  } catch (err) {
    alert('删除失败: ' + (err.message || '未知错误'))
  }
}

const openAddModal = () => {
  Object.assign(currentAssistant, {
    id: '',
    name: '',
    description: '',
    prompt: '',
  })
  isModalOpen.value = true
}

const closeModal = () => {
  isModalOpen.value = false
}

const saveAssistant = async () => {
  try {
    const payload = { ...currentAssistant }

    if (currentAssistant.id) {
      await assistantApi.updateById(currentAssistant.id, payload)
    } else {
      payload.gmt_create = new Date().toISOString().replace('T', ' ')
      await assistantApi.save(payload)
    }
    
    await fetchAssistants()
    const targetId = currentAssistant.id || sortedAssistants.value[0]?.id
    if (targetId) {
      const targetAssistant = assistants.value.find(a => a.id === targetId)
      if (targetAssistant) handleSelectAssistant(targetAssistant)
    }
    closeModal()
  } catch (err) {
    alert('保存失败: ' + (err.message || '未知错误'))
  }
}

const handleResetHistory = async () => {
  if (!selectedAssistantId.value) return
  if (!confirm('确定要重置该助手的所有对话记录吗？')) return
  
  try {
    await historyApi.resetByAssistantId(selectedAssistantId.value)
    loadingHistory.value = true
    const res = await historyApi.getByAssistantId(selectedAssistantId.value)
    historyMessages.value = res.data?.messages || res.messages || []
    await nextTick()
    scrollToBottom(true)
  } catch (err) {
    alert('重置失败: ' + (err.message || '未知错误'))
  } finally {
    loadingHistory.value = false
  }
}

// 生成唯一ID（仅前端用于v-for的key，不传给后端）
const generateUniqueId = () => {
  return `${Date.now()}-${Math.floor(Math.random() * 10000)}`
}

const sendMessage = async () => {
  if (!selectedAssistantId.value) {
    alert('请先从左侧选择一个助手');
    return;
  }
  
  const inputText = userInput.value.trim();
  if (!inputText) return;
  
  // 中止当前请求（优化：彻底清除第一条消息的处理状态）
  if (sending.value && streamAbortController.value) {
    // 1. 标记第一条消息为已中止
    const lastIndex = historyMessages.value.length - 1;
    if (lastIndex >= 0 && historyMessages.value[lastIndex].isLoading) {
      historyMessages.value[lastIndex] = {
        ...historyMessages.value[lastIndex],
        output: { content: historyMessages.value[lastIndex].output.content + '（已中止）' },
        isLoading: false,
        isAborted: true // 添加明确的中止标记
      };
      historyMessages.value = [...historyMessages.value];
      await nextTick();
    }
    
    // 2. 中止请求并清除控制器
    streamAbortController.value.abort();
    streamAbortController.value = null;
    sending.value = false; // 确保状态重置
    await new Promise(resolve => setTimeout(resolve, 200)); // 等待中止完成
  }
  
  // 初始化第二条消息的状态（确保与第一条消息完全隔离）
  userInput.value = '';
  sending.value = true;
  
  const currentAssist = assistants.value.find(a => a.id === selectedAssistantId.value);
  if (!currentAssist) {
    alert('未找到当前助手信息');
    sending.value = false;
    return;
  }
  
  // 构建请求数据
  const requestData = {
    prompt: currentAssist.prompt,
    send: inputText
  };
  
  // 添加用户消息（使用新的ID和独立状态）
  const userMessageId = generateUniqueId();
  const userMessage = {
    id: userMessageId,
    input: requestData,
    output: { content: '' },
    usage: { input_tokens: 0, output_tokens: 0, total_tokens: 0 },
    isUser: true
  };
  historyMessages.value.push(userMessage);
  await nextTick();
  scrollToBottom(true);
  
  // 添加加载消息（使用新的ID和独立状态）
  const loadingMessageId = generateUniqueId();
  const loadingMessage = {
    id: loadingMessageId,
    input: { prompt: '', send: '' },
    output: { content: '' },
    usage: { input_tokens: 0, output_tokens: 0, total_tokens: 0 },
    isLoading: true,
    isAssistant: true
  };
  historyMessages.value.push(loadingMessage);
  await nextTick();
  scrollToBottom(true);

  // 创建新的AbortController（与第一条消息完全隔离）
  const newAbortController = new AbortController();
  streamAbortController.value = newAbortController;
  const signal = newAbortController.signal;
  
  // 为第二条消息创建独立的内容变量（关键：避免与第一条消息共享）
  let secondMessageContent = '';
  
  try {
    await historyApi.streamProcessMessage(
      selectedAssistantId.value,
      requestData,
      signal,
      
      // 接收流式内容（只更新当前消息）
      (content) => {
        secondMessageContent += content;
        // 找到当前加载消息的索引（通过ID精准定位，避免索引混淆）
        const currentIndex = historyMessages.value.findIndex(msg => msg.id === loadingMessageId);
        if (currentIndex !== -1) {
          historyMessages.value[currentIndex] = {
            ...historyMessages.value[currentIndex],
            output: { content: secondMessageContent },
            isLoading: true
          };
          historyMessages.value = [...historyMessages.value]; // 强制更新
          nextTick(() => autoScroll.value && scrollToBottom());
        }
      },
      
      // 完成回调（只更新当前消息）
      (usage) => {
        sending.value = false;
        streamAbortController.value = null;
        
        // 通过ID精准定位当前消息
        const currentIndex = historyMessages.value.findIndex(msg => msg.id === loadingMessageId);
        if (currentIndex !== -1) {
          historyMessages.value[currentIndex] = {
            ...historyMessages.value[currentIndex],
            output: { content: secondMessageContent },
            usage: usage || { input_tokens: 0, output_tokens: 0, total_tokens: 0 },
            isLoading: false
          };
          historyMessages.value = [...historyMessages.value];
          nextTick(() => scrollToBottom(true));
        }
      }
    );
  } catch (err) {
    sending.value = false;
    streamAbortController.value = null;
    
    // 通过ID精准定位当前消息
    const currentIndex = historyMessages.value.findIndex(msg => msg.id === loadingMessageId);
    if (currentIndex !== -1 && historyMessages.value[currentIndex].isLoading) {
      if (err.name === 'AbortError') {
        historyMessages.value[currentIndex] = {
          ...historyMessages.value[currentIndex],
          output: { content: secondMessageContent + '（已中止）' },
          isLoading: false
        };
      } else {
        historyMessages.value.splice(currentIndex, 1); // 移除错误消息
      }
      historyMessages.value = [...historyMessages.value];
    }
  }
};

const stopMessage = async () => {
  if (!sending.value || !streamAbortController.value) return;

  const lastIndex = historyMessages.value.length - 1;
  if (lastIndex >= 0 && historyMessages.value[lastIndex].isLoading) {
    // 立即更新为中止状态
    historyMessages.value[lastIndex] = {
      ...historyMessages.value[lastIndex],
      output: { content: historyMessages.value[lastIndex].output.content + '（已中止）' },
      isLoading: false
    };
    historyMessages.value = [...historyMessages.value]; // 强制更新
    await nextTick();
  }
  
  streamAbortController.value.abort();
  streamAbortController.value = null;
  sending.value = false;
};

const scrollToBottom = async (force = false) => {
  const historyContainer = document.querySelector('.history-container')
  if (!historyContainer) return
  
  if (!force) {
    const scrollBottom = historyContainer.scrollHeight - historyContainer.scrollTop;
    const isAtBottom = scrollBottom <= historyContainer.clientHeight + 50;
    if (!isAtBottom) {
      autoScroll.value = false;
      return;
    }
  }
  
  await nextTick();
  
  const lastMessage = document.querySelector('.message-item:last-child');
  if (lastMessage) {
    lastMessage.scrollIntoView({ behavior: 'smooth', block: 'end' });
  } else {
    historyContainer.scrollTop = historyContainer.scrollHeight;
  }
  
  autoScroll.value = true;
}

// 生命周期
onMounted(() => {
  fetchAssistants();
  
  const historyContainer = document.querySelector('.history-container');
  if (historyContainer) {
    historyContainer.addEventListener('scroll', () => {
      clearTimeout(scrollTimeout);
      isScrolling.value = true;
      
      scrollTimeout = setTimeout(() => {
        isScrolling.value = false;
        const scrollBottom = historyContainer.scrollHeight - historyContainer.scrollTop;
        autoScroll.value = scrollBottom <= historyContainer.clientHeight + 50;
      }, 300);
    });
  }
})

onUnmounted(() => {
  const historyContainer = document.querySelector('.history-container');
  if (historyContainer) {
    historyContainer.removeEventListener('scroll', () => {});
  }
  if (streamAbortController.value) {
    streamAbortController.value.abort();
    streamAbortController.value = null;
  }
})

watch(() => historyMessages.value.length, async (newLen) => {
  if (newLen > 0 && autoScroll.value && !sending.value) {
    await nextTick();
    scrollToBottom();
  }
})
</script>

<style scoped>
.container {
  display: flex;
  width: 100vw;
  height: 100vh;
  margin: 0;
  padding: 0;
  background-color: #f3f4f6;
  overflow: hidden;
}

.right-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

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
</style>