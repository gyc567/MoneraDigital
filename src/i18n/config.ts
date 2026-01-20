import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import en from './locales/en.json?raw';
import zh from './locales/zh.json?raw';

// Parse JSON strings to objects
let enData: Record<string, unknown>;
let zhData: Record<string, unknown>;
try {
  enData = JSON.parse(en);
  zhData = JSON.parse(zh);
  console.log('[i18n] Translation files loaded successfully');
  console.log('[i18n] en.json keys:', Object.keys(enData).length);
  console.log('[i18n] zh.json keys:', Object.keys(zhData).length);
} catch (error) {
  console.error('[i18n] Failed to parse translation files:', error);
  enData = {};
  zhData = {};
}

// 获取保存在localStorage中的语言偏好，默认为英文
const getSavedLanguage = (): string => {
  const saved = localStorage.getItem('i18n-language');
  return saved && ['en', 'zh'].includes(saved) ? saved : 'en';
};

// i18n配置
const resources = {
  en: { translation: enData },
  zh: { translation: zhData },
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
    debug: true, // Enable debug mode
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
  localStorage.setItem('i18n-language', lng);
});

export default i18n;
