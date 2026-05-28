export function showToast(title) {
  uni.showToast({ title, icon: 'none', duration: 1800 })
}

export function navToTab(url) {
  uni.switchTab({ url })
}
