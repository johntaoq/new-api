/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { normalizeLanguage, supportedLanguages } from './language';

const translationLoaders = {
  en: () => import('./locales/en.json'),
  fr: () => import('./locales/fr.json'),
  'zh-CN': () => import('./locales/zh-CN.json'),
  'zh-TW': () => import('./locales/zh-TW.json'),
  ru: () => import('./locales/ru.json'),
  ja: () => import('./locales/ja.json'),
  vi: () => import('./locales/vi.json'),
};

const translationBackend = {
  type: 'backend',
  init() {},
  read(language, _namespace, callback) {
    const normalizedLanguage = normalizeLanguage(language);
    const loader =
      translationLoaders[normalizedLanguage] || translationLoaders['zh-CN'];

    loader()
      .then((module) => callback(null, module.default || module))
      .catch((error) => callback(error, false));
  },
};

export const i18nReady = i18n
  .use(LanguageDetector)
  .use(translationBackend)
  .use(initReactI18next)
  .init({
    load: 'currentOnly',
    supportedLngs: supportedLanguages,
    fallbackLng: 'zh-CN',
    nsSeparator: false,
    interpolation: {
      escapeValue: false,
    },
  });

if (typeof window !== 'undefined') {
  window.__i18n = i18n;
}

export default i18n;
