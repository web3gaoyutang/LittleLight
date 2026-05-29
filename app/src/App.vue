<script>
export default {}
</script>

<style>
page {
  min-height: 100%;
  background:
    linear-gradient(130deg, rgba(231,242,255,.88) 0%, rgba(249,244,255,.74) 33%, rgba(255,249,237,.82) 66%, rgba(239,252,248,.78) 100%),
    linear-gradient(180deg, #fbfcff 0%, #f6f8ff 46%, #fff8f2 100%);
  color: #14313b;
  font-family: -apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", sans-serif;
}
button,
uni-button {
  appearance: none;
  -webkit-appearance: none;
  border: none;
  margin-left: 0 !important;
  margin-right: 0 !important;
  padding-left: 0;
  padding-right: 0;
  overflow: visible;
  background: transparent;
  line-height: normal;
}

uni-button.primary-btn,
uni-button.ghost-btn,
uni-button.avatar-btn,
uni-button.wechat-btn,
uni-button.dev-btn,
uni-button.role-toggle,
.primary-btn,
.ghost-btn,
.avatar-btn {
  margin-left: 0 !important;
  margin-right: 0 !important;
}
button::after,
uni-button::after {
  display: none !important;
  border: none !important;
}
button[disabled],
uni-button[disabled] { opacity: .58; }

uni-page-head,
.uni-page-head,
.uni-placeholder {
  display: none !important;
}

uni-tabbar.uni-tabbar-bottom {
  left: 50% !important;
  right: auto !important;
  bottom: calc(12px + env(safe-area-inset-bottom, 0px)) !important;
  width: min(calc(100% - 24px), 370px) !important;
  transform: translateX(-50%) !important;
  border-radius: 30px !important;
  overflow: visible !important;
  background: transparent !important;
}

uni-tabbar .uni-tabbar {
  height: 68px !important;
  padding: 9px !important;
  box-sizing: border-box !important;
  border-radius: 30px !important;
  border: 1px solid rgba(255,255,255,.78) !important;
  background: linear-gradient(145deg, rgba(255,255,255,.78), rgba(255,255,255,.50)) !important;
  box-shadow: 0 1px 0 rgba(255,255,255,.86) inset, 0 18px 42px rgba(73,91,146,.15) !important;
  backdrop-filter: blur(26px) saturate(1.22) !important;
}

uni-tabbar .uni-tabbar::before {
  content: "";
  position: absolute;
  inset: -22px -12px -14px;
  z-index: -1;
  pointer-events: none;
  border-radius: 38px;
  background: linear-gradient(180deg, rgba(250,250,255,0), rgba(250,250,255,.82) 48%, rgba(250,250,255,.96));
}

uni-tabbar .uni-tabbar-border {
  display: none !important;
}

uni-tabbar .uni-tabbar__item {
  min-width: 0 !important;
  padding: 0 2px !important;
  -webkit-tap-highlight-color: transparent !important;
}

uni-tabbar .uni-tabbar__bd {
  height: 50px !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  border-radius: 22px !important;
  background: transparent !important;
  box-shadow: none !important;
}

uni-tabbar .uni-tabbar__label {
  margin-top: 0 !important;
  display: flex !important;
  flex-direction: column !important;
  align-items: center !important;
  justify-content: center !important;
  gap: 2px !important;
  font-size: 11px !important;
  line-height: 1.15 !important;
  font-weight: 900 !important;
  color: #767a96 !important;
}

uni-tabbar .uni-tabbar__item:has(.uni-tabbar__label[style*="63, 90, 148"]) .uni-tabbar__label {
  color: #3f5a94 !important;
}

uni-tabbar .uni-tabbar__item:active .uni-tabbar__bd,
uni-tabbar .uni-tabbar__item:focus .uni-tabbar__bd,
uni-tabbar .uni-tabbar__item:hover .uni-tabbar__bd {
  background: transparent !important;
  box-shadow: none !important;
}

uni-tabbar .uni-tabbar__label::before {
  width: 28px;
  height: 28px;
  display: block;
  border-radius: 999px;
  padding: 4px;
  box-sizing: border-box;
  background-color: currentColor;
  background-image: none;
  mask-repeat: no-repeat;
  mask-position: center;
  mask-size: 18px 18px;
  -webkit-mask-repeat: no-repeat;
  -webkit-mask-position: center;
  -webkit-mask-size: 18px 18px;
  box-shadow: none;
  transition: transform .16s ease, background-color .16s ease;
}

uni-tabbar .uni-tabbar__item:has(.uni-tabbar__label[style*="63, 90, 148"]) .uni-tabbar__label::before {
  color: #3f5a94;
  background-color: currentColor;
  filter: drop-shadow(0 6px 10px rgba(63,90,148,.16));
}

uni-tabbar .uni-tabbar__item:active .uni-tabbar__label::before {
  transform: scale(.94);
}

uni-tabbar .uni-tabbar__item:nth-of-type(2) .uni-tabbar__label::before {
  content: "";
  mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M4 10.5 12 4l8 6.5'/%3E%3Cpath d='M6.5 10.5V20h11v-9.5'/%3E%3Cpath d='M10 20v-5h4v5'/%3E%3C/svg%3E");
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M4 10.5 12 4l8 6.5'/%3E%3Cpath d='M6.5 10.5V20h11v-9.5'/%3E%3Cpath d='M10 20v-5h4v5'/%3E%3C/svg%3E");
}

uni-tabbar .uni-tabbar__item:nth-of-type(3) .uni-tabbar__label::before {
  content: "";
  mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M20.5 8.7c0 5.2-8.5 10.1-8.5 10.1S3.5 13.9 3.5 8.7A4.6 4.6 0 0 1 12 6.2a4.6 4.6 0 0 1 8.5 2.5Z'/%3E%3C/svg%3E");
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M20.5 8.7c0 5.2-8.5 10.1-8.5 10.1S3.5 13.9 3.5 8.7A4.6 4.6 0 0 1 12 6.2a4.6 4.6 0 0 1 8.5 2.5Z'/%3E%3C/svg%3E");
}

uni-tabbar .uni-tabbar__item:nth-of-type(4) .uni-tabbar__label::before {
  content: "";
  mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Crect x='5' y='7' width='14' height='12' rx='4'/%3E%3Cpath d='M12 7V4'/%3E%3Cpath d='M9 4h6'/%3E%3Cpath d='M9 15h6'/%3E%3Ccircle cx='9' cy='12' r='.6'/%3E%3Ccircle cx='15' cy='12' r='.6'/%3E%3C/svg%3E");
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Crect x='5' y='7' width='14' height='12' rx='4'/%3E%3Cpath d='M12 7V4'/%3E%3Cpath d='M9 4h6'/%3E%3Cpath d='M9 15h6'/%3E%3Ccircle cx='9' cy='12' r='.6'/%3E%3Ccircle cx='15' cy='12' r='.6'/%3E%3C/svg%3E");
}

uni-tabbar .uni-tabbar__item:nth-of-type(5) .uni-tabbar__label::before {
  content: "";
  mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Crect x='4' y='5' width='16' height='15' rx='3'/%3E%3Cpath d='M8 3v4'/%3E%3Cpath d='M16 3v4'/%3E%3Cpath d='M4 10h16'/%3E%3C/svg%3E");
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Crect x='4' y='5' width='16' height='15' rx='3'/%3E%3Cpath d='M8 3v4'/%3E%3Cpath d='M16 3v4'/%3E%3Cpath d='M4 10h16'/%3E%3C/svg%3E");
}

uni-tabbar .uni-tabbar__item:nth-of-type(6) .uni-tabbar__label::before {
  content: "";
  mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Ccircle cx='12' cy='8' r='4'/%3E%3Cpath d='M5 20a7 7 0 0 1 14 0'/%3E%3C/svg%3E");
  -webkit-mask-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round' xmlns='http://www.w3.org/2000/svg'%3E%3Ccircle cx='12' cy='8' r='4'/%3E%3Cpath d='M5 20a7 7 0 0 1 14 0'/%3E%3C/svg%3E");
}
</style>
