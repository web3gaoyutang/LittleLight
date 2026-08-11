export const DICTATION_TEXT_EVENT = 'littlelight:dictation-text'

export function currentRoutePath() {
  const pages = getCurrentPages?.() || []
  const current = pages[pages.length - 1]
  return current?.route || ''
}

export function dictationTargetForRoute(route = currentRoutePath()) {
  if (route.includes('/communication/')) return 'communication'
  if (route.includes('/schedule/')) return 'schedule'
  if (route.includes('/heal/') || route.includes('/home/')) return 'healing'
  return 'draft'
}

export function emitDictationText(text, target = dictationTargetForRoute()) {
  const payload = { text: String(text || '').trim(), target, route: currentRoutePath() }
  if (!payload.text) return false
  // #ifdef H5
  window.dispatchEvent(new CustomEvent(DICTATION_TEXT_EVENT, { detail: payload }))
  // #endif
  uni.$emit(DICTATION_TEXT_EVENT, payload)
  return true
}

export function onDictationText(handler) {
  const wrapped = (event) => handler(event?.detail || event)
  // #ifdef H5
  window.addEventListener(DICTATION_TEXT_EVENT, wrapped)
  // #endif
  uni.$on(DICTATION_TEXT_EVENT, wrapped)
  return () => {
    // #ifdef H5
    window.removeEventListener(DICTATION_TEXT_EVENT, wrapped)
    // #endif
    uni.$off(DICTATION_TEXT_EVENT, wrapped)
  }
}

export function appendDictationText(current, text) {
  const base = String(current || '').trim()
  const next = String(text || '').trim()
  if (!next) return current || ''
  if (!base) return next
  const separator = /[。！？.!?，,；;：:\n]$/.test(base) ? '\n' : '\n'
  return `${base}${separator}${next}`
}

export function createDictationSocket(url) {
  // #ifdef H5
  return createBrowserDictationSocket(url)
  // #endif
  // #ifndef H5
  return createUniDictationSocket(url)
  // #endif
}

export function createPCMRecorder(options = {}) {
  // #ifdef H5
  return new H5PCMRecorder(options)
  // #endif
  // #ifndef H5
  return new UniAppPCMChunkRecorder(options)
  // #endif
}

export function createBrowserDictationSocket(url) {
  const socket = new WebSocket(url)
  const handlers = createSocketHandlers()
  socket.onopen = (event) => handlers.open.forEach((handler) => handler(event))
  socket.onmessage = (event) => handlers.message.forEach((handler) => handler(event))
  socket.onerror = (event) => handlers.error.forEach((handler) => handler(event))
  socket.onclose = (event) => handlers.close.forEach((handler) => handler(event))
  return {
    onOpen(handler) {
      handlers.open.push(handler)
    },
    onMessage(handler) {
      handlers.message.push(handler)
    },
    onError(handler) {
      handlers.error.push(handler)
    },
    onClose(handler) {
      handlers.close.push(handler)
    },
    send(payload) {
      if (socket.readyState !== WebSocket.OPEN) return false
      socket.send(payload)
      return true
    },
    close() {
      if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
        socket.close()
      }
    }
  }
}

export function createUniDictationSocket(url) {
  const task = uni.connectSocket({ url, complete: () => {} })
  const handlers = createSocketHandlers()
  task.onOpen((event) => handlers.open.forEach((handler) => handler(event)))
  task.onMessage((event) => handlers.message.forEach((handler) => handler({ data: event.data })))
  task.onError((event) => handlers.error.forEach((handler) => handler(event)))
  task.onClose((event) => handlers.close.forEach((handler) => handler(event)))
  return {
    onOpen(handler) {
      handlers.open.push(handler)
    },
    onMessage(handler) {
      handlers.message.push(handler)
    },
    onError(handler) {
      handlers.error.push(handler)
    },
    onClose(handler) {
      handlers.close.push(handler)
    },
    send(payload) {
      task.send({ data: payload })
      return true
    },
    close() {
      task.close({})
    }
  }
}

function createSocketHandlers() {
  return {
    open: [],
    message: [],
    error: [],
    close: []
  }
}

export class H5PCMRecorder {
  constructor({ sampleRate = 16000, frameSamples = 640, onFrame } = {}) {
    this.targetSampleRate = sampleRate
    this.frameSamples = frameSamples
    this.onFrame = onFrame
    this.stream = null
    this.audioContext = null
    this.source = null
    this.processor = null
    this.pending = []
    this.pendingLength = 0
  }

  async start() {
    // #ifndef H5
    throw new Error('当前平台暂不支持实时麦克风听写，请先在 H5 端使用。')
    // #endif
    // #ifdef H5
    if (!navigator.mediaDevices?.getUserMedia) {
      throw new Error('当前浏览器不支持麦克风录音')
    }
    this.stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        channelCount: 1,
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true
      }
    })
    const AudioContextClass = window.AudioContext || window.webkitAudioContext
    this.audioContext = new AudioContextClass()
    this.source = this.audioContext.createMediaStreamSource(this.stream)
    this.processor = this.audioContext.createScriptProcessor(4096, 1, 1)
    this.processor.onaudioprocess = (event) => {
      const input = event.inputBuffer.getChannelData(0)
      const downsampled = downsampleBuffer(input, this.audioContext.sampleRate, this.targetSampleRate)
      this.enqueue(downsampled)
    }
    this.source.connect(this.processor)
    this.processor.connect(this.audioContext.destination)
    // #endif
  }

  stop() {
    if (this.processor) {
      this.processor.disconnect()
      this.processor.onaudioprocess = null
      this.processor = null
    }
    if (this.source) {
      this.source.disconnect()
      this.source = null
    }
    if (this.audioContext) {
      this.audioContext.close?.()
      this.audioContext = null
    }
    if (this.stream) {
      this.stream.getTracks().forEach((track) => track.stop())
      this.stream = null
    }
    this.pending = []
    this.pendingLength = 0
  }

  enqueue(float32) {
    if (!float32?.length || typeof this.onFrame !== 'function') return
    this.pending.push(float32)
    this.pendingLength += float32.length
    while (this.pendingLength >= this.frameSamples) {
      const frame = new Float32Array(this.frameSamples)
      let offset = 0
      while (offset < this.frameSamples && this.pending.length) {
        const chunk = this.pending[0]
        const take = Math.min(chunk.length, this.frameSamples - offset)
        frame.set(chunk.subarray(0, take), offset)
        offset += take
        if (take === chunk.length) {
          this.pending.shift()
        } else {
          this.pending[0] = chunk.subarray(take)
        }
        this.pendingLength -= take
      }
      this.onFrame(floatTo16BitPCMBase64(frame))
    }
  }
}

export class UniAppPCMChunkRecorder {
  constructor({ sampleRate = 16000, duration = 280, frameBytes = 1280, frameInterval = 40, onFrame, onError } = {}) {
    this.sampleRate = sampleRate
    this.duration = duration
    this.frameBytes = frameBytes
    this.frameInterval = frameInterval
    this.onFrame = onFrame
    this.onError = onError
    this.manager = null
    this.running = false
    this.recording = false
    this.stopping = false
    this.restartTimer = null
    this.startResolver = null
    this.startRejecter = null
    this.stopResolver = null
    this.audioQueue = Promise.resolve()
    this.stopProcessing = null
  }

  async start() {
    // #ifdef H5
    throw new Error('当前平台请使用浏览器实时录音通道')
    // #endif
    // #ifndef H5
    if (!uni.getRecorderManager) {
      throw new Error('当前 App 版本不支持系统录音能力')
    }
    this.manager = uni.getRecorderManager()
    this.manager.offStart?.()
    this.manager.offStop?.()
    this.manager.offError?.()
    this.running = true
    this.manager.onStart(() => {
      this.recording = true
      this.startResolver?.()
      this.clearStartHandlers()
    })
    this.manager.onStop((event) => this.handleStop(event))
    this.manager.onError((error) => {
      this.recording = false
      const detail = error?.errMsg || '录音启动失败，请检查麦克风权限'
      this.startRejecter?.(new Error(detail))
      this.clearStartHandlers()
      this.fail(detail)
    })
    await this.startChunk()
    // #endif
  }

  stop() {
    this.running = false
    this.stopping = true
    if (this.restartTimer) {
      clearTimeout(this.restartTimer)
      this.restartTimer = null
    }
    if (this.manager && this.recording) {
      return new Promise((resolve) => {
        this.stopResolver = resolve
        const fallback = setTimeout(() => this.resolveStop(), 1200)
        const originalResolve = this.stopResolver
        this.stopResolver = () => {
          clearTimeout(fallback)
          originalResolve?.()
        }
        try {
          this.manager.stop()
        } catch (error) {
          this.resolveStop()
        }
      })
    }
    this.startResolver?.()
    this.recording = false
    this.clearStartHandlers()
    if (this.stopProcessing) {
      return this.stopProcessing.finally(() => this.resolveStop())
    }
    this.resolveStop()
    return Promise.resolve()
  }

  startChunk() {
    return new Promise((resolve, reject) => {
      if (!this.running || !this.manager) {
        resolve()
        return
      }
      this.startResolver = resolve
      this.startRejecter = reject
      this.manager.start({
        duration: this.duration,
        sampleRate: this.sampleRate,
        numberOfChannels: 1,
        encodeBitRate: 256000,
        format: 'PCM'
      })
    })
  }

  async handleStop(event) {
    const processing = this.processStoppedChunk(event)
    this.stopProcessing = processing
    try {
      await processing
    } finally {
      if (this.stopProcessing === processing) {
        this.stopProcessing = null
      }
    }
    if (this.stopping) {
      this.resolveStop()
    }
  }

  async processStoppedChunk(event) {
    this.recording = false
    const tempFilePath = event?.tempFilePath
    const shouldSend = (this.running || this.stopping) && tempFilePath && typeof this.onFrame === 'function'
    const shouldRestart = this.running
    if (shouldRestart) {
      this.restartTimer = setTimeout(() => {
        this.restartTimer = null
        this.startChunk().catch((error) => this.fail(error?.message || '录音重启失败'))
      }, 20)
    }
    if (!shouldSend) return
    try {
      const buffer = await readTempFileAsArrayBuffer(tempFilePath)
      if (buffer?.byteLength) {
        this.audioQueue = this.audioQueue.then(() => this.emitPCMFrames(buffer))
        await this.audioQueue
      }
    } catch (error) {
      this.fail(error?.message || '读取录音片段失败')
    }
  }

  async emitPCMFrames(buffer) {
    const bytes = new Uint8Array(buffer)
    for (let offset = 0; offset < bytes.length; offset += this.frameBytes) {
      if (!this.running && !this.stopping) return
      const frame = bytes.slice(offset, offset + this.frameBytes)
      if (!frame.byteLength) continue
      this.onFrame(arrayBufferToBase64(frame.buffer))
      if (offset + this.frameBytes < bytes.length) {
        await delay(this.frameInterval)
      }
    }
  }

  clearStartHandlers() {
    this.startResolver = null
    this.startRejecter = null
  }

  fail(message) {
    this.running = false
    this.stopping = false
    this.onError?.(message)
    this.resolveStop()
  }

  resolveStop() {
    this.stopping = false
    this.stopResolver?.()
    this.stopResolver = null
  }
}

function downsampleBuffer(buffer, sourceRate, targetRate) {
  if (targetRate === sourceRate) return new Float32Array(buffer)
  if (targetRate > sourceRate) return new Float32Array(buffer)
  const ratio = sourceRate / targetRate
  const length = Math.round(buffer.length / ratio)
  const result = new Float32Array(length)
  let offsetResult = 0
  let offsetBuffer = 0
  while (offsetResult < result.length) {
    const nextOffsetBuffer = Math.round((offsetResult + 1) * ratio)
    let accumulator = 0
    let count = 0
    for (let i = offsetBuffer; i < nextOffsetBuffer && i < buffer.length; i++) {
      accumulator += buffer[i]
      count += 1
    }
    result[offsetResult] = count ? accumulator / count : 0
    offsetResult += 1
    offsetBuffer = nextOffsetBuffer
  }
  return result
}

function floatTo16BitPCMBase64(float32) {
  const buffer = new ArrayBuffer(float32.length * 2)
  const view = new DataView(buffer)
  for (let i = 0; i < float32.length; i++) {
    const sample = Math.max(-1, Math.min(1, float32[i]))
    view.setInt16(i * 2, sample < 0 ? sample * 0x8000 : sample * 0x7fff, true)
  }
  return arrayBufferToBase64(buffer)
}

export function arrayBufferToBase64(buffer) {
  if (typeof uni !== 'undefined' && typeof uni.arrayBufferToBase64 === 'function') {
    return uni.arrayBufferToBase64(buffer)
  }
  if (typeof Buffer !== 'undefined') {
    return Buffer.from(buffer).toString('base64')
  }
  let binary = ''
  const bytes = new Uint8Array(buffer)
  const chunkSize = 0x8000
  for (let i = 0; i < bytes.length; i += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize))
  }
  return btoa(binary)
}

export function readTempFileAsArrayBuffer(filePath) {
  return new Promise((resolve, reject) => {
    const fs = uni.getFileSystemManager?.()
    if (fs?.readFile) {
      fs.readFile({
        filePath,
        success: (res) => resolve(toArrayBuffer(res.data)),
        fail: reject
      })
      return
    }
    // #ifdef APP-PLUS
    readPlusFileAsArrayBuffer(filePath).then(resolve).catch(reject)
    // #endif
    // #ifndef APP-PLUS
    reject(new Error('当前平台无法读取录音片段'))
    // #endif
  })
}

function toArrayBuffer(data) {
  if (data instanceof ArrayBuffer) return data
  if (data?.buffer instanceof ArrayBuffer) {
    return data.buffer.slice(data.byteOffset || 0, (data.byteOffset || 0) + data.byteLength)
  }
  if (typeof data === 'string') {
    if (data.startsWith('data:')) {
      data = data.slice(data.indexOf(',') + 1)
    }
    if (typeof uni !== 'undefined' && typeof uni.base64ToArrayBuffer === 'function') {
      return uni.base64ToArrayBuffer(data)
    }
    const binary = atob(data)
    const buffer = new ArrayBuffer(binary.length)
    const bytes = new Uint8Array(buffer)
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
    return buffer
  }
  return new ArrayBuffer(0)
}

function readPlusFileAsArrayBuffer(filePath) {
  return new Promise((resolve, reject) => {
    const plusGlobal = typeof plus !== 'undefined' ? plus : null
    if (!plusGlobal?.io?.resolveLocalFileSystemURL) {
      reject(new Error('当前 App 运行环境无法读取录音片段'))
      return
    }
    plusGlobal.io.resolveLocalFileSystemURL(filePath, (entry) => {
      entry.file((file) => {
        const reader = new plusGlobal.io.FileReader()
        reader.onloadend = (event) => resolve(toArrayBuffer(event.target.result))
        reader.onerror = reject
        reader.readAsDataURL(file)
      }, reject)
    }, reject)
  })
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
