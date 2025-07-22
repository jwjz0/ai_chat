import request from '@/utils/request'

export default {
    getByAssistantId: (assistantId) => request.get(`/api/voice-robot/v1/history/${assistantId}`),
    resetByAssistantId: (assistantId) => request.delete(`/api/voice-robot/v1/history/${assistantId}`),
    saveByAssistantId: (assistantId, data) => request.post(`/api/voice-robot/v1/history/${assistantId}`, data),

    // 修复后的streamProcessMessage函数
streamProcessMessage(assistantId, data, onMessage, onError, onComplete) {
    const controller = new AbortController();
    const url = `/api/voice-robot/v1/history/${assistantId}/stream-process`;
    let isCompleted = false; // 标记请求是否已完成
    let isAborted = false;   // 标记请求是否被中止
    
    // 新增：存储reader引用，用于强制取消
    let reader = null;
    
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

            reader = response.body.getReader(); // 保存reader引用
            const decoder = new TextDecoder();
            let buffer = '';

            while (true) {
                // 关键改进：在每次循环开始检查中止状态
                if (isAborted) {
                    console.log('检测到中止标志，退出流循环');
                    if (reader) reader.cancel(); // 强制取消读取
                    break;
                }
                
                const { done, value } = await reader.read();
                
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
                                    onError(`解析最后数据错误: ${err.message}`);
                                }
                            }
                        });
                    }
                    
                    // 流结束，触发完成回调
                    if (!isCompleted) {
                        onComplete?.({ input_tokens: 0, output_tokens: 0, total_tokens: 0 });
                        isCompleted = true;
                    }
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
                                isCompleted = true;
                                if (reader) reader.cancel(); // 主动取消读取
                                return;
                            }
                        } catch (err) {
                            onError(`解析错误: ${err.message}`);
                        }
                    }
                });
            }
        } catch (err) {
            // 处理中止错误
            if (err.name === 'AbortError') {
                console.log('请求被中止:', err);
                isAborted = true;
                if (!isCompleted) {
                    onError('请求已取消', { isAborted: true });
                    isCompleted = true;
                }
                return;
            }
            
            // 处理其他错误
            console.error('流式处理错误:', err);
            if (!isCompleted) {
                onError(`流式处理错误: ${err.message}`);
                isCompleted = true;
            }
        } finally {
            // 确保清理资源
            reader = null;
            console.log('流处理最终状态:', { isCompleted, isAborted });
        }
    };

    // 启动流式读取
    fetchStream();

    // 返回取消函数（增强版）
    return () => {
        console.log('尝试中止请求...');
        
        if (isCompleted || isAborted) {
            console.log('请求已完成或已中止，无需操作');
            return;
        }
        
        // 标记为已中止
        isAborted = true;
        
        // 双重保障：先尝试优雅地取消
        try {
            controller.abort();
            console.log('已发送中止信号');
        } catch (err) {
            console.error('中止控制器错误:', err);
        }
        
        // 如果reader存在，强制取消
        if (reader) {
            try {
                reader.cancel();
                console.log('已强制取消reader');
            } catch (err) {
                console.error('取消reader错误:', err);
            }
        }
    };
}
}