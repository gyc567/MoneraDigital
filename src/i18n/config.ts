import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import en from './locales/en.json';
import zh from './locales/zh.json';

console.log('[i18n] Translation files loaded successfully');
console.log('[i18n] en.json keys:', Object.keys(en).length);
console.log('[i18n] zh.json keys:', Object.keys(zh).length);

// Safe localStorage access with fallback
const safeGetItem = (key: string): string | null => {
  try {
    if (typeof window !== 'undefined' && window.localStorage) {
      return window.localStorage.getItem(key);
    }
  } catch (e) {
    console.warn('[i18n] localStorage access failed:', e);
  }
  return null;
};

const safeSetItem = (key: string, value: string): void => {
  try {
    if (typeof window !== 'undefined' && window.localStorage) {
      window.localStorage.setItem(key, value);
    }
  } catch (e) {
    console.warn('[i18n] localStorage write failed:', e);
  }
};

// 获取保存在localStorage中的语言偏好，默认为英文
const getSavedLanguage = (): string => {
  const saved = safeGetItem('i18n-language');
  return saved && ['en', 'zh'].includes(saved) ? saved : 'en';
};

// i18n配置
const resources = {
  en: { translation: en },
  zh: { translation: zh },
};

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: getSavedLanguage(),
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false, // React已经防止XSS
    },
    debug: import.meta.env.DEV, // Only enable debug in development
  })
  .then(() => {
    console.log('[i18n] Initialization complete');
    console.log('[i18n] Current language:', i18n.language);
  })
  .catch((error: Error) => {
    console.error('[i18n] Initialization failed:', error);
  });

// 监听语言变化，保存到localStorage
i18n.on('languageChanged', (lng: string) => {
  console.log('[i18n] Language changed to:', lng);
  safeSetItem('i18n-language', lng);
});

export default i18n;
