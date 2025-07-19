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
          :disabled="!selectedAssistantId || sending"
          :sending="sending"
          @input-change="userInput = $event"
          @send="sendMessage"
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

// 助手列表数据
const assistants = ref([])
const loading = ref(false)
const error = ref('')
const selectedAssistantId = ref('')
const assistantRefs = ref({})

// 当前选中的助手
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

// 处理键盘事件
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
  Object.assign(currentAssistant, {
    id: '',
    name: '',
    description: '',
    prompt: '',
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

    if (currentAssistant.id) {
      await assistantApi.updateById(currentAssistant.id, payload)
    } else {
      payload.gmt_create = now
      await assistantApi.save(payload)
    }
    
    await fetchAssistants()
    const targetId = currentAssistant.id || sortedAssistants.value[0]?.id
    if (targetId) {
      // 选中新创建/编辑的助手
      const targetAssistant = assistants.value.find(a => a.id === targetId)
      if (targetAssistant) handleSelectAssistant(targetAssistant)
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
    
    await nextTick()
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

// 滚动到历史底部
const scrollToBottom = async (force = false) => {
  const historyContainer = document.querySelector('.history-container');
  if (!historyContainer) return;
  
  if (!force) {
    const scrollBottom = historyContainer.scrollHeight - historyContainer.scrollTop;
    const isAtBottom = scrollBottom <= historyContainer.clientHeight + 50;
    
    if (!isAtBottom) {
      autoScroll.value = false;
      return;
    }
  }
  
  await nextTick();
  
  const lastMessage = document.getElementById(`msg-${historyData.messages.length - 1}`);
  if (lastMessage) {
    lastMessage.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  } else {
    historyContainer.scrollTop = historyContainer.scrollHeight;
  }
  
  autoScroll.value = true;
}

// 初始化
onMounted(() => {
  fetchAssistants();
  
  // 设置滚动监听
  const historyContainer = document.querySelector('.history-container');
  if (historyContainer) {
    historyContainer.addEventListener('scroll', () => {
      if (scrollTimeout) clearTimeout(scrollTimeout);
      
      isScrolling.value = true;
      
      scrollTimeout = setTimeout(() => {
        isScrolling.value = false;
        
        const scrollBottom = historyContainer.scrollHeight - historyContainer.scrollTop;
        const isAtBottom = scrollBottom <= historyContainer.clientHeight + 50;
        
        if (isAtBottom) {
          autoScroll.value = true;
        }
      }, 300);
    });
  }
  
  nextTick(() => {
    scrollToBottom(true);
  });
});

// 组件卸载时取消请求
onUnmounted(() => {
  cancelStream()
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
</style>