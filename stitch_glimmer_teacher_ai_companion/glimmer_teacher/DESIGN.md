---
name: Glimmer Teacher
colors:
  surface: '#f9f9f7'
  surface-dim: '#dadad8'
  surface-bright: '#f9f9f7'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f4f4f2'
  surface-container: '#eeeeec'
  surface-container-high: '#e8e8e6'
  surface-container-highest: '#e2e3e1'
  on-surface: '#1a1c1b'
  on-surface-variant: '#464554'
  inverse-surface: '#2f3130'
  inverse-on-surface: '#f1f1ef'
  outline: '#777585'
  outline-variant: '#c7c4d6'
  surface-tint: '#4e4cce'
  primary: '#4441c4'
  on-primary: '#ffffff'
  primary-container: '#5d5cde'
  on-primary-container: '#f1eeff'
  inverse-primary: '#c2c1ff'
  secondary: '#7d5700'
  on-secondary: '#ffffff'
  secondary-container: '#ffc55f'
  on-secondary-container: '#755100'
  tertiary: '#51517d'
  on-tertiary: '#ffffff'
  tertiary-container: '#696997'
  on-tertiary-container: '#f2eeff'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#e2dfff'
  primary-fixed-dim: '#c2c1ff'
  on-primary-fixed: '#0b006b'
  on-primary-fixed-variant: '#3530b6'
  secondary-fixed: '#ffdeaa'
  secondary-fixed-dim: '#f5bd58'
  on-secondary-fixed: '#271900'
  on-secondary-fixed-variant: '#5f4100'
  tertiary-fixed: '#e2dfff'
  tertiary-fixed-dim: '#c3c2f6'
  on-tertiary-fixed: '#171640'
  on-tertiary-fixed-variant: '#43426e'
  background: '#f9f9f7'
  on-background: '#1a1c1b'
  surface-variant: '#e2e3e1'
typography:
  display-lg:
    fontFamily: Manrope
    fontSize: 48px
    fontWeight: '700'
    lineHeight: 60px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Manrope
    fontSize: 32px
    fontWeight: '700'
    lineHeight: 44px
    letterSpacing: -0.01em
  headline-lg-mobile:
    fontFamily: Manrope
    fontSize: 28px
    fontWeight: '700'
    lineHeight: 38px
  headline-md:
    fontFamily: Manrope
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
  title-lg:
    fontFamily: Manrope
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
  body-lg:
    fontFamily: Manrope
    fontSize: 18px
    fontWeight: '400'
    lineHeight: 28px
  body-md:
    fontFamily: Manrope
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  label-md:
    fontFamily: Manrope
    fontSize: 14px
    fontWeight: '500'
    lineHeight: 20px
    letterSpacing: 0.01em
  caption:
    fontFamily: Manrope
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 16px
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  base: 4px
  xs: 8px
  sm: 12px
  md: 16px
  lg: 24px
  xl: 32px
  2xl: 48px
  gutter: 20px
  margin-mobile: 16px
  margin-desktop: 40px
---

## 品牌与风格

此设计系统的核心理念是“专业、宁静、共情”。它专为教育者设计，旨在创造一个高品质、低压力的数字工作空间，传达出一种如微光般温暖而坚定的专业力量。

设计风格融合了**现代极简主义 (Minimalism)** 与 **触感设计 (Tactile Design)**。通过大量的留白、精致的排版以及细腻的色彩分层，规避传统教育产品的低质感或过度紧凑感。视觉语言应引发用户的心理安全感和职业成就感，平衡“工具的精准”与“教育的人文关怀”。

## 色彩

本系统严格弃用任何绿色系，转而采用以“暮光紫”为主轴的调色盘，营造专业且静谧的氛围。

- **暮光紫 (Primary):** #5D5CDE。作为核心品牌色，用于主按钮、导航状态和关键行动点。它比纯蓝更具人文气息，比纯紫更显稳重。
- **暮光浅紫 (Tertiary):** #8E8DBE。用于次要交互、辅助图标或背景叠加，平滑视觉过渡。
- **暖金 (Secondary/Accent):** #D9A441。作为“治愈”与“高光”的象征，用于表扬勋章、重要提醒和强调元素，模拟微光的温暖。
- **暖灰中性色 (Neutrals):** 
  - 底色层：#F7F7F5 (Off-white)，提供比纯白更柔和、更高级的视觉底色。
  - 描边/分割：#E5E5E0，保持界面的结构感但不突兀。
  - 文字主色：#2D2C33，深紫色调的黑，替代纯黑以保持色调统一。

## 字体排印

全系统选用 **Manrope**。这款字体兼具几何感与人文细节，在中文环境下通过优化字间距与行高，能够展现出极佳的现代感与可读性。

- **对比度:** 严格区分标题与正文的字重。大标题使用 Bold (700) 配合负数间距营造高级感；正文使用 Regular (400) 并保持充足的行高 (1.5x - 1.6x)，确保长时间阅读不疲劳。
- **层次结构:** 
  - 重点信息（如学生姓名、课程标题）统一使用 `title-lg`。
  - 辅助说明（如时间戳、备注）使用 `label-md` 或 `caption`。
- **多语言适配:** 确保 Manrope 的数字与英文在中文句子中表现自然，基线对齐严谨。

## 布局与间距

此设计系统采用 **8像素栅格系统** 作为基准，但在精细处允许 4px 的微调，以追求更加紧致的高端视觉效果。

- **布局模型:** 
  - 桌面端：采用 12 列响应式网格，侧边导航栏固定，内容区最大宽度 1440px。
  - 移动端：采用单列流式布局，侧边安全边距为 16px。
- **间距逻辑:** 
  - 组件内部（如卡片内的头像与标题）：使用 `sm` (12px)。
  - 组件之间：使用 `lg` (24px) 或 `xl` (32px)，通过充裕的留白来降低认知负载。
- **分段:** 页面大块功能区间使用 `2xl` (48px) 进行物理隔离，强化层级分明感。

## 层级与深度

为了实现“暮光”般的深邃感，本系统摒弃传统的高强度阴影，改用**色调图层 (Tonal Layers)** 与 **微弱环境光阴影**：

1.  **基础层:** 采用背景色 #F7F7F5，无阴影。
2.  **内容卡片:** 背景为纯白 (#FFFFFF)，带有极浅的、色彩倾向于主色调的阴影（Offset: 0, 4px; Blur: 20px; Color: rgba(93, 92, 222, 0.04)）。
3.  **浮动组件 (如下拉菜单/弹窗):** 使用更明显的深度感，并配合 1px 的超细浅灰紫色边框 (#EBEBFA) 增强轮廓定义，而非单纯依靠阴影。
4.  **微光特效:** 在关键数据卡片下，可使用极低透明度的暮光紫 (#5D5CDE) 进行背景光晕处理，营造一种“光从屏幕深处透出”的质感。

## 形状

全系统采用统一的 **12px (0.75rem)** 圆角逻辑，这在感官上比直角更具亲和力，比全圆角更显专业与理智。

- **标准圆角 (Rounded-MD):** 12px。用于主要按钮、输入框、课程卡片。
- **大圆角 (Rounded-LG):** 16px。用于浮动模态框、大型容器。
- **全圆角 (Pill):** 用于状态标签（Chips）或头像，打破网格的严肃感，增加活力。
- **描边:** 组件描边厚度统一为 1px，避免使用粗边框，以维持高端的轻盈感。

## 组件

1.  **按钮 (Buttons):**
    - **主按钮:** 背景 #5D5CDE，文字纯白，12px 圆角。悬停时色值加深 10%。
    - **治愈按钮 (Heal Action):** 背景 #D9A441，用于“点赞”或“发送鼓励”，带有微弱的金色外发光。
2.  **输入框 (Inputs):**
    - 采用填充式设计。默认状态背景为 #FFFFFF，带有 1px #E5E5E0 边框；聚焦状态边框变为 #5D5CDE，并增加淡紫色光晕。
3.  **课程卡片 (Cards):**
    - 纯白底色，12px 圆角。标题使用 `title-lg`，底部辅助信息使用 `label-md` 配色 #8E8DBE。
4.  **进度条 (Progress Bars):**
    - 轨道使用淡灰色 #F0F0EE，填充部分使用 #5D5CDE。对于已完成或满分状态，可切换为 #D9A441 以示奖励。
5.  **反馈系统 (Feedback):**
    - 提示消息 (Toasts) 采用深紫色背景（#2D2C33），文字白色，放置在页面顶部中央，通过高对比度引导注意，但不破坏底层暖色调的和谐。