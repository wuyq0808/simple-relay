import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import en from './locales/en.json';
import zh from './locales/zh.json';

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: {
        translation: en
      },
      zh: {
        translation: zh
      }
    },
    fallbackLng: 'en',
    debug: false,
    
    interpolation: {
      escapeValue: false
    },

    detection: {
      order: ['navigator', 'localStorage', 'cookie'],
      lookupLocalStorage: 'i18nextLng',
      lookupCookie: 'i18next',
      caches: ['localStorage', 'cookie']
    }
  });

export default i18n;