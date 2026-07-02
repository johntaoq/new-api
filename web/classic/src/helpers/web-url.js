import { encodeToBase64 } from './base64';

const trimTrailingSlash = (value = '') => value.replace(/\/+$/, '');

const normalizeApiKey = (key = '') => {
  if (!key) {
    return '';
  }
  return key.startsWith('sk-') ? key : `sk-${key}`;
};

const getRootBaseUrl = (serverAddress = '') =>
  trimTrailingSlash(serverAddress || window.location.origin);

const getOpenAIBaseUrl = (serverAddress = '') =>
  `${getRootBaseUrl(serverAddress)}/v1`;

const encode = (value) => encodeURIComponent(value);

const buildProviderConfig = (serverAddress, key) => {
  const rootBaseUrl = getRootBaseUrl(serverAddress);
  const openAIBaseUrl = getOpenAIBaseUrl(serverAddress);
  const apiKey = normalizeApiKey(key);

  return {
    id: 'new-api',
    provider: 'new-api',
    apiKey,
    baseUrl: openAIBaseUrl,
    rootBaseUrl,
    chat: {
      apiKey,
      baseUrl: openAIBaseUrl,
      completionsUrl: `${openAIBaseUrl}/chat/completions`,
    },
    images: {
      apiKey,
      baseUrl: openAIBaseUrl,
      generationsUrl: `${openAIBaseUrl}/images/generations`,
      editsUrl: `${openAIBaseUrl}/images/edits`,
    },
  };
};

const replaceWebUrlPlaceholders = (url, config) => {
  const openWebUIConfig = encodeToBase64(JSON.stringify(config));
  const replacements = {
    '{address}': encode(config.rootBaseUrl),
    '{rawAddress}': config.rootBaseUrl,
    '{baseUrl}': encode(config.baseUrl),
    '{baseURL}': encode(config.baseUrl),
    '{rawBaseUrl}': config.baseUrl,
    '{rawBaseURL}': config.baseUrl,
    '{openaiBaseUrl}': encode(config.baseUrl),
    '{openaiBaseURL}': encode(config.baseUrl),
    '{rawOpenAIBaseUrl}': config.baseUrl,
    '{rawOpenAIBaseURL}': config.baseUrl,
    '{key}': config.apiKey,
    '{apiKey}': config.apiKey,
    '{chatCompletionsUrl}': encode(config.chat.completionsUrl),
    '{rawChatCompletionsUrl}': config.chat.completionsUrl,
    '{imageGenerationUrl}': encode(config.images.generationsUrl),
    '{imageGenerationsUrl}': encode(config.images.generationsUrl),
    '{rawImageGenerationUrl}': config.images.generationsUrl,
    '{rawImageGenerationsUrl}': config.images.generationsUrl,
    '{imageEditUrl}': encode(config.images.editsUrl),
    '{imageEditsUrl}': encode(config.images.editsUrl),
    '{rawImageEditUrl}': config.images.editsUrl,
    '{rawImageEditsUrl}': config.images.editsUrl,
    '{newApiConfig}': encode(openWebUIConfig),
    '{openWebUIConfig}': encode(openWebUIConfig),
  };

  return Object.entries(replacements).reduce(
    (nextUrl, [placeholder, value]) => nextUrl.replaceAll(placeholder, value),
    url,
  );
};

const appendOpenWebUIParams = (url, config) => {
  try {
    const parsed = new URL(url);
    const target = `${parsed.hostname}${parsed.pathname}`.toLowerCase();
    if (
      !target.includes('webui') &&
      !target.includes('open-webui') &&
      parsed.port !== '7000'
    ) {
      return url;
    }

    parsed.searchParams.set('new_api_base_url', config.baseUrl);
    parsed.searchParams.set('new_api_api_key', config.apiKey);
    parsed.searchParams.set(
      'new_api_chat_completions_url',
      config.chat.completionsUrl,
    );
    parsed.searchParams.set(
      'new_api_image_generations_url',
      config.images.generationsUrl,
    );
    parsed.searchParams.set('new_api_image_edits_url', config.images.editsUrl);
    parsed.searchParams.set(
      'new_api_config',
      encodeToBase64(JSON.stringify(config)),
    );
    return parsed.toString();
  } catch {
    return url;
  }
};

export const buildExternalWebUrl = ({ url, key, serverAddress }) => {
  if (!url || !serverAddress || !key) {
    return '';
  }

  const config = buildProviderConfig(serverAddress, key);
  let nextUrl = url;

  if (nextUrl.includes('{cherryConfig}')) {
    const cherryConfig = {
      id: 'new-api',
      baseUrl: config.rootBaseUrl,
      apiKey: config.apiKey,
    };
    nextUrl = nextUrl.replaceAll(
      '{cherryConfig}',
      encode(encodeToBase64(JSON.stringify(cherryConfig))),
    );
  }

  if (nextUrl.includes('{aionuiConfig}')) {
    const aionuiConfig = {
      platform: 'new-api',
      baseUrl: config.rootBaseUrl,
      apiKey: config.apiKey,
    };
    nextUrl = nextUrl.replaceAll(
      '{aionuiConfig}',
      encode(encodeToBase64(JSON.stringify(aionuiConfig))),
    );
  }

  nextUrl = replaceWebUrlPlaceholders(nextUrl, config);
  return appendOpenWebUIParams(nextUrl, config);
};
