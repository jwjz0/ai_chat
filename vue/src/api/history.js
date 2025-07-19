import request from '@/utils/request'

export default {
    getByAssistantId: (assistantId) => request.get(`/api/voice-robot/v1/history/${assistantId}`),
    resetByAssistantId: (assistantId) => request.delete(`/api/voice-robot/v1/history/${assistantId}`),
    saveByAssistantId: (assistantId, data) => request.post(`/api/voice-robot/v1/history/${assistantId}`, data),

    streamProcessMessage(assistantId, data, onMessage, onError, onComplete) {
    const controller = new AbortController();
    const url = `/api/voice-robot/v1/history/${assistantId}/stream-process`;
    let isCompleted = false; // 标记是否已触发完成回调

    const fetchStream = async () => {
        try {
        const response = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data),
            signal: controller.signal,
        });

        if (!response.ok) {
            throw new Error(`请求失败: ${response.status}`);
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
            const { done, value } = await reader.read();
            
            // 处理流结束的情况（没有更多数据）
            if (done) {
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
                    onError(`解析最后数据错误: ${err.message}`);
                    }
                }
                });
            }
            
            // 流结束，强制触发完成回调（确保前端停止加载）
            if (!isCompleted) {
                onComplete?.({ input_tokens: 0, output_tokens: 0, total_tokens: 0 });
                isCompleted = true;
            }
            break; // 退出循环
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
                    onComplete?.(data.usage);
                    isCompleted = true;
                    reader.cancel(); // 主动取消读取，加速结束
                    return;
                }
                } catch (err) {
                onError(`解析错误: ${err.message}`);
                }
            }
            });
        }
        } catch (err) {
        if (!isCompleted) { // 避免在已完成后重复触发错误
            if (err.name === 'AbortError') {
            onError('请求已取消');
            } else {
            onError(`流式处理错误: ${err.message}`);
            }
        }
        }
    };

    // 启动流式读取
    fetchStream();

    // 返回取消函数
    return () => {
        if (!isCompleted) {
        controller.abort();
        }
    };
    }
}