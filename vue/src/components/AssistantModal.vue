<template>
  <div v-if="visible" class="modal-overlay" @click="$emit('close')">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h3>{{ assistant.id ? '编辑助手' : '新增助手' }}</h3>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>
      
      <div class="modal-body">
        <form @submit.prevent="handleSave">
          <div class="form-group">
            <label>名称 <span class="required">*</span></label>
            <input
              type="text"
              v-model="localAssistant.name"
              required
              placeholder="请输入助手名称"
            >
          </div>
          
          <div class="form-group">
            <label>描述</label>
            <textarea
              v-model="localAssistant.description"
              placeholder="请输入助手描述"
              rows="2"
            ></textarea>
          </div>
          
          <div class="form-group">
            <label>提示词 <span class="required">*</span></label>
            <textarea
              v-model="localAssistant.prompt"
              required
              placeholder="请输入提示词"
              rows="5"
            ></textarea>
          </div>
          
          <div class="form-actions">
            <button type="button" class="cancel-btn" @click="$emit('close')">取消</button>
            <button type="submit" class="save-btn">保存</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';

const props = defineProps({
  visible: { type: Boolean, default: false },
  assistant: { 
    type: Object, 
    default: () => ({
      id: '',
      name: '',
      description: '',
      prompt: ''
    })
  }
});

const emit = defineEmits(['close', 'save']);

const localAssistant = ref({
  id: '',
  name: '',
  description: '',
  prompt: ''
});

watch(() => props.visible, (newVal) => {
  if (newVal) {
    localAssistant.value = { ...props.assistant };
  }
});

const handleSave = () => {
  // 严格验证并清理数据
  const name = (localAssistant.value.name || '').trim();
  const description = (localAssistant.value.description || '').trim();
  const prompt = (localAssistant.value.prompt || '').trim();

  if (!name) {
    alert('请输入助手名称');
    return;
  }
  
  if (!prompt) {
    alert('请输入提示词');
    return;
  }

  // 构建严格符合后端要求的纯净数据结构
  const payload = {
    name,
    description,
    prompt
    // 不包含任何其他字段（如id、时间戳等）
  };

  // 提交数据：分离后端结构体和前端逻辑ID
  emit('save', {
    payload,         // 后端需要的结构体
    id: localAssistant.value.id  // 仅用于前端逻辑
  });
};
</script>

<style scoped>
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
</style>