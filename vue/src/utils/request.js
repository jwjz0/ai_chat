import axios from 'axios'

const request = axios.create({
  baseURL: 'http://localhost:8080',
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json;charset=utf-8'
  }
})

request.interceptors.response.use(
  (response) => {
    if (response.config.responseType === 'blob') {
      return response;
    }

    const res = response.data
    if (res.code !== 200) {
      console.error('接口错误:', res.message || '请求失败')
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return res.data
  },
  (error) => {
    if (axios.isCancel(error)) {
      console.log('请求已取消:', error.message)
      return Promise.reject(new Error('请求已取消'))
    }

    let errorMsg = '网络异常，请稍后重试'
    if (error.response) {
      switch (error.response.status) {
        case 401:
          errorMsg = '未授权，请重新登录'
          break
        case 403:
          errorMsg = '权限不足'
          break
        case 404:
          errorMsg = '接口不存在'
          break
        case 500:
          errorMsg = '服务器内部错误'
          break
        default:
          errorMsg = `请求错误 (${error.response.status})`
      }
    } else if (error.message.includes('timeout')) {
      errorMsg = '请求超时，请重试'
    }

    console.error('请求错误:', errorMsg)
    return Promise.reject(new Error(errorMsg))
  }
)

export default request