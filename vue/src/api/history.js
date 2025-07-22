import request from '@/utils/request'

export default {
    getByAssistantId: (assistantId) => request.get(`/api/voice-robot/v1/history/${assistantId}`),
    resetByAssistantId: (assistantId) => request.delete(`/api/voice-robot/v1/history/${assistantId}`),
    saveByAssistantId: (assistantId, data) => request.post(`/api/voice-robot/v1/history/${assistantId}`, data),

    async streamProcessMessage(assistantId, data, signal, onMessage, onComplete) {
    const url = `/api/voice-robot/v1/history/${assistantId}/stream-process`;
    
    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
        signal: signal,
      });

      if (!response.ok) {
        throw new Error(`请求失败: ${response.status} ${response.statusText}`);
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        // 使用Promise.race添加超时处理
        const { done, value } = await Promise.race([
          reader.read(),
          new Promise((_, reject) => 
            setTimeout(() => reject(new Error('读取超时')), 30000)
          )
        ]);
        
        // 处理流结束的情况
        if (done) {
          console.log('流处理完成');
          // 处理最后残留的缓冲数据
          if (buffer.trim()) {
            const lines = buffer.split('\n');
            lines.forEach(line => {
              line = line.trim();
              if (line.startsWith('data: ')) {
                const dataStr = line.slice(5);
                try {
                  const data = JSON.parse(dataStr);
                  if (data.content) {
                    onMessage(data.content);
                  }
                } catch (err) {
                  console.error('解析最后数据错误:', err);
                }
              }
            });
          }
          
          // 流结束，触发完成回调
          onComplete?.({ input_tokens: 0, output_tokens: 0, total_tokens: 0 });
          break;
        }

        // 处理流式数据
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || ''; // 保留不完整的行到下一轮处理

        lines.forEach(line => {
          line = line.trim();
          if (!line) return;

          if (line.startsWith('data: ')) {
            const dataStr = line.slice(5);
            try {
              const data = JSON.parse(dataStr);
              if (data.content) {
                onMessage(data.content);
              } else if (data.done) {
                // 收到后端明确的完成信号
                console.log('收到后端完成信号');
                onComplete?.(data.usage);
                return;
              }
            } catch (err) {
              console.error('解析错误:', err);
            }
          }
        });
      }
    } catch (err) {
      // 重新抛出错误，让调用者处理
      throw err;
    }
  }
}