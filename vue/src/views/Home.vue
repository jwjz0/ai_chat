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
        <ChatHistory 
          :assistant="selectedAssistant"
          :messages="historyData.messages"
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
import ChatHistory from '@/components/ChatHistory.vue'
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

const historyData = reactive({ messages: [] })
const loadingHistory = ref(false)

const userInput = ref('')
const sending = ref(false)
const streamController = ref(null)

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
  return historyData.messages?.reduce((sum, msg) => {
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
  if (e.shiftKey) {
    const cursorPos = e.target.selectionStart;
    userInput.value = userInput.value.substring(0, cursorPos) + '\n' + userInput.value.substring(cursorPos);
    nextTick(() => {
      e.target.selectionStart = e.target.selectionEnd = cursorPos + 1;
    });
  } else {
    sendMessage();
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

const handleSelectAssistant = async (assistant) => {
  selectedAssistantId.value = assistant.id
  historyData.messages = []
  loadingHistory.value = true
  
  try {
    const res = await historyApi.getByAssistantId(assistant.id)
    historyData.messages = res.data?.messages || res.messages || []
    await nextTick()
    setTimeout(() => {
      scrollToBottom(true)
    }, 100)
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
      historyData.messages = []
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
    historyData.messages = res.data?.messages || res.messages || []
    await nextTick()
    scrollToBottom(true)
  } catch (err) {
    alert('重置失败: ' + (err.message || '未知错误'))
  } finally {
    loadingHistory.value = false
  }
}

const sendMessage = async () => {
  // 验证输入
  if (!selectedAssistantId.value) {
    alert('请先从左侧选择一个助手');
    return;
  }
  
  const inputText = userInput.value.trim();
  if (!inputText) return;
  
  // 立即清空输入框并更新发送状态
  userInput.value = '';
  sending.value = true;
  
  // 获取当前选中的助手信息
  const currentAssist = assistants.value.find(a => a.id === selectedAssistantId.value);
  if (!currentAssist) {
    alert('未找到当前助手信息');
    sending.value = false;
    return;
  }
  
  // 添加用户消息到历史记录
  const userMessage = {
    input: { prompt: currentAssist.prompt, send: inputText },
    output: { content: '' },
    usage: { input_tokens: 0, output_tokens: 0, total_tokens: 0 },
  };
  
  historyData.messages.push(userMessage);
  await scrollToBottom(true);
  
  // 添加加载中的消息占位符
  const loadingMessage = {
    input: { prompt: '', send: '' },
    output: { content: '' },
    usage: { input_tokens: 0, output_tokens: 0, total_tokens: 0 },
    isLoading: true
  };
  
  historyData.messages.push(loadingMessage);
  await scrollToBottom(true);

  // 用于累积流式响应内容
  let fullContent = '';
  
  // 调用API发起流式请求
  streamController.value = historyApi.streamProcessMessage(
    selectedAssistantId.value,
    { prompt: currentAssist.prompt, send: inputText },
    
    // 处理流式响应内容
    (content) => {
      fullContent += content;
      
      // 更新消息内容和状态
      historyData.messages[historyData.messages.length - 1] = {
        ...loadingMessage,
        output: { content: fullContent },
        isLoading: true
      };
      
      // 自动滚动到底部
      nextTick(() => {
        if (autoScroll.value) {
          scrollToBottom();
        }
      });
    },
    
    // 处理错误（包括中止）
    (errorMsg, options = {}) => {
      console.log('流式请求错误:', errorMsg, options);
      
      // 无论如何都结束发送状态
      sending.value = false;
      
      if (options.isAborted) {
        // 用户主动中止：更新消息为“已中断”
        historyData.messages[historyData.messages.length - 1] = {
          ...loadingMessage,
          output: { content: fullContent + '（已中断）' },
          isLoading: false
        };
      } else {
        // 其他错误：提示用户并移除加载状态
        alert(errorMsg);
        historyData.messages.pop(); // 移除加载中的消息
      }
    },
    
    // 处理完成回调
    (usage) => {
      // 计算token使用量（示例：简单基于字符数估算）
      const userInputTokens = Math.ceil(inputText.length / 4);
      const assistantOutputTokens = Math.ceil(fullContent.length / 4);
      const totalTokens = userInputTokens + assistantOutputTokens;

      // 更新用户消息的token使用量
      historyData.messages[historyData.messages.length - 2].usage = {
        input_tokens: userInputTokens,
        output_tokens: 0,
        total_tokens: totalTokens
      };

      // 更新助手回复的内容和token使用量
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

      // 强制更新消息数组以触发视图更新
      historyData.messages = [...historyData.messages];
      
      // 结束发送状态
      sending.value = false;
      
      // 滚动到底部
      nextTick(() => scrollToBottom(true));
    }
  );
};

const stopMessage = () => {
    console.log('用户点击停止按钮');
    
    if (!sending.value || !streamController.value) {
        console.log('无需中止：sending=', sending.value, 'controller=', streamController.value);
        return;
    }
    
    // 立即更新状态，防止重复点击
    sending.value = false;
    
    // 调用中止函数
    try {
        streamController.value();
        console.log('已调用中止函数');
    } catch (err) {
        console.error('中止函数调用错误:', err);
    } finally {
        streamController.value = null;
    }
};

const scrollToBottom = async (force = false) => {
  const historyContainer = document.querySelector('.history-container')
  if (!historyContainer) return
  
  if (!force) {
    const scrollBottom = historyContainer.scrollHeight - historyContainer.scrollTop
    const isAtBottom = scrollBottom <= historyContainer.clientHeight + 50
    
    if (!isAtBottom) {
      autoScroll.value = false
      return
    }
  }
  
  await nextTick()
  
  const lastMessage = document.getElementById(`msg-${historyData.messages.length - 1}`)
  if (lastMessage) {
    lastMessage.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
  } else {
    historyContainer.scrollTop = historyContainer.scrollHeight
  }
  
  autoScroll.value = true
}

// 生命周期
onMounted(() => {
  fetchAssistants()
  
  const historyContainer = document.querySelector('.history-container')
  if (historyContainer) {
    historyContainer.addEventListener('scroll', () => {
      if (scrollTimeout) clearTimeout(scrollTimeout)
      
      isScrolling.value = true
      
      scrollTimeout = setTimeout(() => {
        isScrolling.value = false
        
        const scrollBottom = historyContainer.scrollHeight - historyContainer.scrollTop
        const isAtBottom = scrollBottom <= historyContainer.clientHeight + 50
        
        if (isAtBottom) {
          autoScroll.value = true
        }
      }, 300)
    })
  }
  
  setTimeout(() => {
    scrollToBottom(true)
  }, 300)
})

onUnmounted(() => {
  if (streamController.value) {
    streamController.value.abort()
  }
})

watch(() => historyData.messages.length, async () => {
  if (autoScroll.value && !sending.value) {
    await scrollToBottom()
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