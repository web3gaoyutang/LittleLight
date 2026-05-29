export function showToast(title) {
  uni.showToast({ title: String(title || '操作失败，请稍后重试'), icon: 'none', duration: 1800 })
}

export function confirmAction({ title = '确认操作', content = '确认继续吗？', confirmText = '确认', cancelText = '取消' } = {}) {
  return new Promise((resolve) => {
    uni.showModal({
      title,
      content,
      confirmText,
      cancelText,
      success: (res) => resolve(!!res.confirm),
      fail: () => resolve(false)
    })
  })
}

export function navToTab(url) {
  uni.switchTab({ url })
}

export function errorMessage(error, fallback = '操作失败，请稍后重试') {
  if (error?.unauthorized) return '登录已过期，请重新登录'
  if (error?.rateLimited) {
    const retryAfter = Number(error.retryAfter || 0)
    return retryAfter > 0 ? `请求太频繁，请 ${retryAfter} 秒后再试` : '请求太频繁，请稍后再试'
  }
  return error?.message || error?.detail || error?.error || error?.errMsg || fallback
}

export function ensureLoggedIn(api) {
  if (api.isLoggedIn()) return true
  api.logout?.()
  uni.reLaunch({ url: '/pages/login/index' })
  return false
}

export function trimmed(value) {
  return String(value ?? '').trim()
}

export function hasText(value) {
  return trimmed(value).length > 0
}

export function runeLength(value) {
  return Array.from(String(value ?? '')).length
}

export function withinLength(value, max) {
  return runeLength(value) <= max
}

export function validClock(value) {
  const match = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(trimmed(value))
  return !!match
}

export function clockBefore(start, end) {
  if (!validClock(start) || !validClock(end)) return false
  return trimmed(start) < trimmed(end)
}

export function normalizeFormError(error, fallback) {
  if (!error) return ''
  return errorMessage(error, fallback)
}

export function isHighRiskSafety(value) {
  return ['crisis_support_required', 'student_safety_review_required', 'medical_review_required'].includes(value)
}

export function chooseImportFile({ extensions = ['.xlsx', '.csv'] } = {}) {
  return new Promise((resolve, reject) => {
    const normalizedExtensions = extensions.map((item) => String(item || '').trim().replace(/^\./, '')).filter(Boolean)
    const h5Extensions = normalizedExtensions.map((item) => `.${item}`)
    const done = (file) => {
      const uploadFile = file?.path || file?.file || file?.tempFilePath || file
      if (!uploadFile) {
        reject(new Error('未读取到文件'))
        return
      }
      resolve(uploadFile)
    }
    const fail = (error) => reject(error)

    // #ifdef MP-WEIXIN
    if (typeof uni.chooseMessageFile === 'function') {
      uni.chooseMessageFile({
        count: 1,
        type: 'file',
        extension: normalizedExtensions,
        success: (res) => done(res.tempFiles?.[0]),
        fail
      })
      return
    }
    // #endif

    if (typeof uni.chooseFile === 'function') {
      uni.chooseFile({
        count: 1,
        extension: h5Extensions,
        success: (res) => done(res.tempFiles?.[0]),
        fail
      })
      return
    }

    reject(new Error('当前端不支持选择文件，请先下载模板并在 H5 或微信文件选择器中导入'))
  })
}
