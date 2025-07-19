import request from '@/utils/request'

// 助手管理 API（对应路由：/api/voice-robot/v1/assistant）
export default {
  // 获取所有助手列表
  getAll: () => request.get('/api/voice-robot/v1/assistant'),

  // 根据ID删除助手
  deleteById: (id) => request.delete(`/api/voice-robot/v1/assistant/${id}`),

  // 新增助手（data需包含name、prompt等字段）
  save: (data) => request.post('/api/voice-robot/v1/assistant', data),

  // 根据ID更新助手（data包含需更新的字段）
  updateById: (id, data) => request.put(`/api/voice-robot/v1/assistant/${id}`, data)
}