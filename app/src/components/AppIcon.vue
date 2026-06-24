<template>
  <svg
    class="app-icon"
    viewBox="0 0 24 24"
    fill="none"
    aria-hidden="true"
    focusable="false"
  >
    <g
      stroke="currentColor"
      :stroke-width="strokeWidth"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path v-for="(d, index) in icon.paths || []" :key="`p-${index}`" :d="d" />
      <circle
        v-for="(shape, index) in icon.circles || []"
        :key="`c-${index}`"
        :cx="shape.cx"
        :cy="shape.cy"
        :r="shape.r"
      />
      <rect
        v-for="(shape, index) in icon.rects || []"
        :key="`r-${index}`"
        :x="shape.x"
        :y="shape.y"
        :width="shape.width"
        :height="shape.height"
        :rx="shape.rx"
      />
      <line
        v-for="(shape, index) in icon.lines || []"
        :key="`l-${index}`"
        :x1="shape.x1"
        :y1="shape.y1"
        :x2="shape.x2"
        :y2="shape.y2"
      />
      <polyline
        v-for="(shape, index) in icon.polylines || []"
        :key="`pl-${index}`"
        :points="shape"
      />
      <polygon
        v-for="(shape, index) in icon.polygons || []"
        :key="`pg-${index}`"
        :points="shape"
      />
    </g>
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, default: 'sparkles' },
  strokeWidth: { type: [Number, String], default: 2.1 }
})

const icons = {
  activity: {
    polylines: ['3 12 7 12 9 6 14 18 16 12 21 12']
  },
  alert: {
    paths: ['M12 4.5 21 20H3L12 4.5Z'],
    lines: [
      { x1: 12, y1: 10, x2: 12, y2: 14 },
      { x1: 12, y1: 17, x2: 12.01, y2: 17 }
    ]
  },
  arrowUpRight: {
    paths: ['M7 17 17 7', 'M9 7h8v8']
  },
  bell: {
    paths: ['M6 9a6 6 0 0 1 12 0c0 7 2 7 2 9H4c0-2 2-2 2-9', 'M10 20a2.5 2.5 0 0 0 4 0']
  },
  bot: {
    rects: [{ x: 5, y: 7, width: 14, height: 12, rx: 4 }],
    paths: ['M12 7V4', 'M9 4h6', 'M9 15h6'],
    circles: [
      { cx: 9, cy: 12, r: 0.6 },
      { cx: 15, cy: 12, r: 0.6 }
    ]
  },
  bookmark: {
    paths: ['M6 4h12v17l-6-3.6L6 21V4Z']
  },
  calendar: {
    rects: [{ x: 4, y: 5, width: 16, height: 15, rx: 3 }],
    lines: [
      { x1: 8, y1: 3, x2: 8, y2: 7 },
      { x1: 16, y1: 3, x2: 16, y2: 7 },
      { x1: 4, y1: 10, x2: 20, y2: 10 }
    ]
  },
  calendarCheck: {
    rects: [{ x: 4, y: 5, width: 16, height: 15, rx: 3 }],
    lines: [
      { x1: 8, y1: 3, x2: 8, y2: 7 },
      { x1: 16, y1: 3, x2: 16, y2: 7 },
      { x1: 4, y1: 10, x2: 20, y2: 10 }
    ],
    polylines: ['8 15 11 18 16 13']
  },
  check: {
    polylines: ['4 12.5 9.5 18 20 6']
  },
  chevronRight: {
    polylines: ['9 5 16 12 9 19']
  },
  clock: {
    circles: [{ cx: 12, cy: 12, r: 8 }],
    polylines: ['12 7 12 12 15.5 14']
  },
  cloud: {
    paths: ['M7 18h10a4 4 0 0 0 0-8 6 6 0 0 0-11.3-1.8A4.8 4.8 0 0 0 7 18Z']
  },
  copy: {
    rects: [
      { x: 8, y: 8, width: 11, height: 11, rx: 2 },
      { x: 5, y: 5, width: 11, height: 11, rx: 2 }
    ]
  },
  crown: {
    paths: ['M4 18h16', 'M5 9l4 3 3-6 3 6 4-3v7H5V9Z']
  },
  database: {
    paths: ['M5 7c0-2 14-2 14 0s-14 2-14 0Z', 'M5 7v5c0 2 14 2 14 0V7', 'M5 12v5c0 2 14 2 14 0v-5']
  },
  edit: {
    paths: ['M4 20h5l10-10a2.5 2.5 0 0 0-3.5-3.5L5.5 16.5 4 20Z', 'M14 8l2 2']
  },
  empty: {
    rects: [{ x: 5, y: 6, width: 14, height: 12, rx: 3 }],
    paths: ['M9 6a3 3 0 0 1 6 0', 'M9 12h6']
  },
  export: {
    paths: ['M12 4v11', 'M8 8l4-4 4 4', 'M5 14v5h14v-5']
  },
  file: {
    paths: ['M7 3h7l4 4v14H7V3Z', 'M14 3v5h4']
  },
  heart: {
    paths: ['M20.5 8.7c0 5.2-8.5 10.1-8.5 10.1S3.5 13.9 3.5 8.7A4.6 4.6 0 0 1 12 6.2a4.6 4.6 0 0 1 8.5 2.5Z']
  },
  heartHand: {
    paths: ['M12 20s-7-4.4-7-9a3.8 3.8 0 0 1 7-2.1A3.8 3.8 0 0 1 19 11c0 4.6-7 9-7 9Z', 'M3 18h5l3 2 5-4']
  },
  home: {
    paths: ['M4 10.5 12 4l8 6.5', 'M6.5 10.5V20h11v-9.5', 'M10 20v-5h4v5']
  },
  info: {
    circles: [{ cx: 12, cy: 12, r: 8 }],
    lines: [
      { x1: 12, y1: 11, x2: 12, y2: 16 },
      { x1: 12, y1: 8, x2: 12.01, y2: 8 }
    ]
  },
  listChecks: {
    paths: ['M9 6h11', 'M9 12h11', 'M9 18h11'],
    polylines: ['3 6 4.5 7.5 7 4.8', '3 12 4.5 13.5 7 10.8', '3 18 4.5 19.5 7 16.8']
  },
  loading: {
    paths: ['M12 4a8 8 0 1 0 8 8'],
    lines: [{ x1: 12, y1: 4, x2: 12, y2: 8 }]
  },
  lock: {
    rects: [{ x: 5, y: 10, width: 14, height: 10, rx: 3 }],
    paths: ['M8 10V8a4 4 0 0 1 8 0v2']
  },
  logout: {
    paths: ['M10 5H6v14h4', 'M14 8l4 4-4 4', 'M18 12H9']
  },
  message: {
    paths: ['M5 6.5A4.5 4.5 0 0 1 9.5 2h5A4.5 4.5 0 0 1 19 6.5v4A4.5 4.5 0 0 1 14.5 15H11l-5 4v-4.8A4.5 4.5 0 0 1 5 10.5v-4Z']
  },
  mic: {
    rects: [{ x: 9, y: 3, width: 6, height: 11, rx: 3 }],
    paths: ['M5 11a7 7 0 0 0 14 0', 'M12 18v3', 'M9 21h6']
  },
  mapPin: {
    paths: ['M12 21s6-5.2 6-11a6 6 0 1 0-12 0c0 5.8 6 11 6 11Z'],
    circles: [{ cx: 12, cy: 10, r: 2.2 }]
  },
  moon: {
    paths: ['M19 14.8A7.5 7.5 0 0 1 9.2 5a7 7 0 1 0 9.8 9.8Z']
  },
  pause: {
    lines: [
      { x1: 9, y1: 6, x2: 9, y2: 18 },
      { x1: 15, y1: 6, x2: 15, y2: 18 }
    ]
  },
  play: {
    polygons: ['8 5 19 12 8 19 8 5']
  },
  plus: {
    lines: [
      { x1: 12, y1: 5, x2: 12, y2: 19 },
      { x1: 5, y1: 12, x2: 19, y2: 12 }
    ]
  },
  search: {
    circles: [{ cx: 11, cy: 11, r: 6 }],
    lines: [{ x1: 16, y1: 16, x2: 20, y2: 20 }]
  },
  settings: {
    circles: [{ cx: 12, cy: 12, r: 3 }],
    paths: ['M12 3v3', 'M12 18v3', 'M4.2 7.5l2.6 1.5', 'M17.2 15l2.6 1.5', 'M19.8 7.5 17.2 9', 'M6.8 15l-2.6 1.5']
  },
  shieldCheck: {
    paths: ['M12 3 19 6v5c0 5-3.2 8.4-7 10-3.8-1.6-7-5-7-10V6l7-3Z'],
    polylines: ['8.5 12 11 14.5 15.5 9.5']
  },
  smile: {
    circles: [{ cx: 12, cy: 12, r: 8 }],
    paths: ['M8.5 14.5a5 5 0 0 0 7 0'],
    lines: [
      { x1: 9, y1: 10, x2: 9.01, y2: 10 },
      { x1: 15, y1: 10, x2: 15.01, y2: 10 }
    ]
  },
  sparkles: {
    paths: ['M12 3l1.7 5.1L19 10l-5.3 1.9L12 17l-1.7-5.1L5 10l5.3-1.9L12 3Z', 'M5 15l.8 2.2L8 18l-2.2.8L5 21l-.8-2.2L2 18l2.2-.8L5 15Z']
  },
  square: {
    rects: [{ x: 7, y: 7, width: 10, height: 10, rx: 2 }]
  },
  table: {
    rects: [{ x: 4, y: 5, width: 16, height: 14, rx: 3 }],
    lines: [
      { x1: 4, y1: 10, x2: 20, y2: 10 },
      { x1: 9, y1: 5, x2: 9, y2: 19 },
      { x1: 15, y1: 5, x2: 15, y2: 19 }
    ]
  },
  target: {
    circles: [
      { cx: 12, cy: 12, r: 8 },
      { cx: 12, cy: 12, r: 3 }
    ],
    lines: [{ x1: 12, y1: 2, x2: 12, y2: 5 }]
  },
  trash: {
    paths: ['M5 7h14', 'M9 7V5h6v2', 'M7 7l1 13h8l1-13'],
    lines: [
      { x1: 10, y1: 11, x2: 10, y2: 17 },
      { x1: 14, y1: 11, x2: 14, y2: 17 }
    ]
  },
  upload: {
    paths: ['M12 16V4', 'M8 8l4-4 4 4', 'M5 16v4h14v-4']
  },
  user: {
    circles: [{ cx: 12, cy: 8, r: 4 }],
    paths: ['M5 20a7 7 0 0 1 14 0']
  },
  wechat: {
    paths: ['M10.5 6C6.9 6 4 8.3 4 11.1c0 1.6.9 3 2.4 3.9L6 18l3-1.5c.5.1 1 .2 1.5.2 3.6 0 6.5-2.3 6.5-5.1S14.1 6 10.5 6Z', 'M14 11.5c3 0 5.5 1.9 5.5 4.2 0 1.2-.7 2.3-1.8 3.1L18 21l-2.5-1.2c-.5.1-1 .1-1.5.1-2.6 0-4.8-1.4-5.3-3.2'],
    lines: [
      { x1: 8.5, y1: 10, x2: 8.51, y2: 10 },
      { x1: 12.5, y1: 10, x2: 12.51, y2: 10 }
    ]
  },
  wind: {
    paths: ['M4 8h9a3 3 0 1 0-3-3', 'M4 13h13a3 3 0 1 1-3 3', 'M4 18h6']
  },
  x: {
    paths: ['M6 6l12 12', 'M18 6 6 18']
  }
}

const icon = computed(() => icons[props.name] || icons.sparkles)
</script>

<style scoped>
.app-icon {
  width: 1em;
  height: 1em;
  display: block;
  color: inherit;
}
</style>
