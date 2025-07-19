
/**
 * 滚动到容器底部
 * @param {Ref} containerRef - 容器DOM引用
 * @param {boolean} force - 是否强制滚动（即使不在底部）
 * @returns {Promise<boolean>} 是否执行了滚动
 */
export const scrollToBottom = async (containerRef, force = false) => {
  if (!containerRef.value) return false;
  const container = containerRef.value;

  // 非强制模式下，判断是否已在底部
  if (!force) {
    const scrollBottom = container.scrollHeight - container.scrollTop;
    const isAtBottom = scrollBottom <= container.clientHeight + 50;
    if (!isAtBottom) return false;
  }

  // 等待DOM更新后滚动
  await Promise.resolve();
  container.scrollTop = container.scrollHeight;
  return true;
};

/**
 * 设置滚动监听（用于自动滚动控制）
 * @param {Ref} containerRef - 容器DOM引用
 * @param {Ref<boolean>} autoScrollRef - 自动滚动开关
 * @param {Ref<boolean>} isScrollingRef - 是否正在滚动
 * @returns {Function} 清理监听的函数
 */
export const setupScrollListener = (containerRef, autoScrollRef, isScrollingRef) => {
  if (!containerRef.value) return () => {};

  let scrollTimeout = null;
  const handleScroll = () => {
    if (scrollTimeout) clearTimeout(scrollTimeout);
    isScrollingRef.value = true;

    scrollTimeout = setTimeout(() => {
      isScrollingRef.value = false;
      const container = containerRef.value;
      const scrollBottom = container.scrollHeight - container.scrollTop;
      const isAtBottom = scrollBottom <= container.clientHeight + 50;
      if (isAtBottom) autoScrollRef.value = true;
    }, 300);
  };

  containerRef.value.addEventListener('scroll', handleScroll);
  return () => {
    containerRef.value?.removeEventListener('scroll', handleScroll);
    if (scrollTimeout) clearTimeout(scrollTimeout);
  };
};