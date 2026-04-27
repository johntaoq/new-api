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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Empty,
  InputNumber,
  Select,
  Spin,
  Tag,
  TextArea,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import {
  AlertTriangle,
  Image as ImageIcon,
  Sparkles,
  Download,
  Clock,
  Trash2,
  Upload,
  X,
} from 'lucide-react';
import { API } from '../../helpers';

const { Text, Title } = Typography;
const IMAGE_ENDPOINT_TYPE = 'image-generation';
const IMAGE_HISTORY_STORAGE_KEY = 'image_playground_history';
const MAX_HISTORY_ITEMS = 8;
const MAX_HISTORY_CHARS = 4_000_000;
const IMAGE_REQUEST_TIMEOUT_MS = 610000;

const gptImage2SizeOptions = [
  { label: '1024x1024', value: '1024x1024' },
  { label: '1024x1536', value: '1024x1536' },
  { label: '1536x1024', value: '1536x1024' },
  { label: '2048x2048', value: '2048x2048' },
  { label: '3840x2160', value: '3840x2160' },
  { label: '2160x3840', value: '2160x3840' },
];

const gptImageSizeOptions = [
  { label: '1024x1024', value: '1024x1024' },
  { label: '1024x1536', value: '1024x1536' },
  { label: '1536x1024', value: '1536x1024' },
];

const maiImageSizeOptions = [
  { label: '1024x1024', value: '1024x1024' },
  { label: '1024x768', value: '1024x768' },
  { label: '768x1024', value: '768x1024' },
  { label: '1365x768', value: '1365x768' },
  { label: '768x1365', value: '768x1365' },
];

const defaultSizeOptions = [
  { label: '1024x1024', value: '1024x1024' },
];

const gptImageQualityOptions = [
  { label: '默认', value: 'auto' },
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
];

const imageModelNamePatterns = [
  'dall-e',
  'gpt-image',
  'mai-image',
  'imagen',
  'image',
  'flux',
  'midjourney',
  'stable-diffusion',
  'stable-image',
];

const isImageModelName = (modelName = '') => {
  const normalized = modelName.toLowerCase();
  return imageModelNamePatterns.some((pattern) => normalized.includes(pattern));
};

const isEditCapableModel = (modelName = '') =>
  modelName.toLowerCase().includes('gpt-image');

const getImageModelProfile = (modelName = '') => {
  const normalized = modelName.toLowerCase();
  if (normalized.includes('mai-image')) {
    return {
      kind: 'mai',
      sizeOptions: maiImageSizeOptions,
      qualityOptions: [],
      supportsQuality: false,
      supportsN: false,
      requestShape: 'width-height',
    };
  }
  if (normalized.includes('gpt-image-2')) {
    return {
      kind: 'gpt-image-2',
      sizeOptions: gptImage2SizeOptions,
      qualityOptions: gptImageQualityOptions,
      supportsQuality: true,
      supportsN: true,
      requestShape: 'size',
    };
  }
  if (normalized.includes('gpt-image')) {
    return {
      kind: 'gpt-image',
      sizeOptions: gptImageSizeOptions,
      qualityOptions: gptImageQualityOptions,
      supportsQuality: true,
      supportsN: true,
      requestShape: 'size',
    };
  }
  return {
    kind: 'default',
    sizeOptions: defaultSizeOptions,
    qualityOptions: [],
    supportsQuality: false,
    supportsN: true,
    requestShape: 'size',
  };
};

const parseImageSize = (value = '1024x1024') => {
  const [width, height] = value.split('x').map((item) => Number(item));
  return {
    width: Number.isFinite(width) ? width : 1024,
    height: Number.isFinite(height) ? height : 1024,
  };
};

const buildImageModelOptions = (models, usableGroup, autoGroups) => {
  if (!Array.isArray(models)) {
    return [];
  }

  const usableGroups = new Set([
    ...Object.keys(usableGroup || {}),
    ...(Array.isArray(autoGroups) ? autoGroups : []),
  ]);

  return models
    .filter((model) => {
      const supportedEndpoints = model.supported_endpoint_types || [];
      const supportsImageEndpoint =
        Array.isArray(supportedEndpoints) &&
        supportedEndpoints.includes(IMAGE_ENDPOINT_TYPE);
      return supportsImageEndpoint || isImageModelName(model.model_name);
    })
    .filter((model) => {
      if (usableGroups.size === 0 || !Array.isArray(model.enable_groups)) {
        return true;
      }
      return model.enable_groups.some((item) => usableGroups.has(item));
    })
    .map((model) => ({
      label: model.model_name,
      value: model.model_name,
      description: model.supported_endpoint_types?.join(', ') || '',
      editCapable: isEditCapableModel(model.model_name),
    }))
    .sort((a, b) => a.value.localeCompare(b.value));
};

const buildFallbackModelOptions = (models) => {
  if (!Array.isArray(models)) {
    return [];
  }
  return models
    .filter(isImageModelName)
    .map((model) => ({
      label: model,
      value: model,
      editCapable: isEditCapableModel(model),
    }))
    .sort((a, b) => a.value.localeCompare(b.value));
};

const getImageSource = (item) => {
  if (item?.b64_json) {
    return `data:image/png;base64,${item.b64_json}`;
  }
  return item?.url || '';
};

const downloadImage = (src, index) => {
  const link = document.createElement('a');
  link.href = src;
  link.download = `new-api-image-${index + 1}.png`;
  document.body.appendChild(link);
  link.click();
  link.remove();
};

const fileToDataUrl = (file) =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result);
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });

const readHistory = () => {
  try {
    const parsed = JSON.parse(
      localStorage.getItem(IMAGE_HISTORY_STORAGE_KEY) || '[]',
    );
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
};

const persistHistory = (items) => {
  let next = items.slice(0, MAX_HISTORY_ITEMS);
  while (
    next.length > 0 &&
    JSON.stringify(next).length > MAX_HISTORY_CHARS
  ) {
    next = next.slice(0, -1);
  }
  localStorage.setItem(IMAGE_HISTORY_STORAGE_KEY, JSON.stringify(next));
  return next;
};

const ImagePlayground = () => {
  const [models, setModels] = useState([]);
  const [groups, setGroups] = useState([]);
  const [model, setModel] = useState('');
  const [group, setGroup] = useState('');
  const [prompt, setPrompt] = useState('');
  const [size, setSize] = useState('1024x1024');
  const [quality, setQuality] = useState('auto');
  const [n, setN] = useState(1);
  const [referenceImages, setReferenceImages] = useState([]);
  const [loadingModels, setLoadingModels] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [elapsedMs, setElapsedMs] = useState(null);
  const [images, setImages] = useState([]);
  const [rawResponse, setRawResponse] = useState(null);
  const [history, setHistory] = useState(() => readHistory());

  const hasModels = models.length > 0;

  const loadModels = useCallback(async () => {
    setLoadingModels(true);
    try {
      const pricingRes = await API.get('/api/pricing', {
        disableDuplicate: true,
      });
      if (pricingRes.data?.success) {
        const options = buildImageModelOptions(
          pricingRes.data.data,
          pricingRes.data.usable_group,
          pricingRes.data.auto_groups,
        );
        setModels(options);
        setModel((current) =>
          options.some((option) => option.value === current)
            ? current
            : options[0]?.value || '',
        );
        return;
      }

      const modelsRes = await API.get('/api/user/models', {
        disableDuplicate: true,
      });
      const fallbackOptions = buildFallbackModelOptions(modelsRes.data?.data);
      setModels(fallbackOptions);
      setModel((current) =>
        fallbackOptions.some((option) => option.value === current)
          ? current
          : fallbackOptions[0]?.value || '',
      );
    } catch (error) {
      Toast.error('加载图片模型失败');
      setModels([]);
      setModel('');
    } finally {
      setLoadingModels(false);
    }
  }, []);

  const loadGroups = useCallback(async () => {
    try {
      const res = await API.get('/api/user/self/groups');
      if (!res.data?.success || !res.data?.data) {
        return;
      }
      const options = Object.entries(res.data.data).map(([value, info]) => ({
        label: info?.desc || value,
        value,
      }));
      setGroups(options);
      setGroup((current) =>
        options.some((option) => option.value === current)
          ? current
          : options[0]?.value || '',
      );
    } catch (error) {
      setGroups([]);
    }
  }, []);

  useEffect(() => {
    loadModels();
    loadGroups();
  }, [loadGroups, loadModels]);

  const selectedModelMeta = useMemo(
    () => models.find((item) => item.value === model),
    [model, models],
  );
  const selectedModelProfile = useMemo(
    () => getImageModelProfile(model),
    [model],
  );
  const canEditSelectedModel = isEditCapableModel(model);
  const isEditRequest = referenceImages.length > 0;

  useEffect(() => {
    if (
      selectedModelProfile.sizeOptions.length > 0 &&
      !selectedModelProfile.sizeOptions.some((option) => option.value === size)
    ) {
      setSize(selectedModelProfile.sizeOptions[0].value);
    }
    if (
      selectedModelProfile.supportsQuality &&
      selectedModelProfile.qualityOptions.length > 0 &&
      !selectedModelProfile.qualityOptions.some(
        (option) => option.value === quality,
      )
    ) {
      setQuality(selectedModelProfile.qualityOptions[0].value);
    }
    if (!selectedModelProfile.supportsQuality && quality !== 'auto') {
      setQuality('auto');
    }
  }, [quality, selectedModelProfile, size]);

  useEffect(() => {
    if (!canEditSelectedModel && referenceImages.length > 0) {
      setReferenceImages([]);
    }
  }, [canEditSelectedModel, referenceImages.length]);

  const handleReferenceUpload = async (event) => {
    const files = Array.from(event.target.files || []);
    event.target.value = '';
    if (files.length === 0) {
      return;
    }

    const imageFiles = files.filter((file) => file.type.startsWith('image/'));
    if (imageFiles.length !== files.length) {
      Toast.warning('只能上传图片文件');
    }

    try {
      const loaded = await Promise.all(
        imageFiles.slice(0, 4).map(async (file) => ({
          name: file.name,
          size: file.size,
          file,
          dataUrl: await fileToDataUrl(file),
        })),
      );
      setReferenceImages((current) => [...current, ...loaded].slice(0, 4));
    } catch (error) {
      Toast.error('读取参考图片失败');
    }
  };

  const removeReferenceImage = (index) => {
    setReferenceImages((current) => current.filter((_, i) => i !== index));
  };

  const clearHistory = () => {
    localStorage.removeItem(IMAGE_HISTORY_STORAGE_KEY);
    setHistory([]);
  };

  const handleGenerate = async () => {
    const trimmedPrompt = prompt.trim();
    if (!model) {
      Toast.warning('请选择图片模型');
      return;
    }
    if (!trimmedPrompt) {
      Toast.warning('请输入提示词');
      return;
    }
    if (referenceImages.length > 0 && !canEditSelectedModel) {
      Toast.warning('当前模型不支持参考图改图，请选择 gpt-image 系列模型');
      return;
    }

    const startedAt = performance.now();
    setGenerating(true);
    setElapsedMs(null);
    setImages([]);
    setRawResponse(null);

    try {
      let res;
      if (referenceImages.length > 0) {
        const formData = new FormData();
        formData.append('model', model);
        formData.append('prompt', trimmedPrompt);
        formData.append('n', String(n));
        formData.append('size', size);
        if (selectedModelProfile.supportsQuality && quality !== 'auto') {
          formData.append('quality', quality);
        }
        if (group) {
          formData.append('group', group);
        }
        const imageFieldName = referenceImages.length > 1 ? 'image[]' : 'image';
        referenceImages.forEach((item) => {
          formData.append(imageFieldName, item.file, item.name);
        });
        res = await API.post('/pg/images/edits', formData, {
          timeout: IMAGE_REQUEST_TIMEOUT_MS,
          skipErrorHandler: true,
        });
      } else {
        const payload = {
          model,
          prompt: trimmedPrompt,
        };
        if (selectedModelProfile.supportsN) {
          payload.n = n;
        }
        if (selectedModelProfile.requestShape === 'width-height') {
          const dimensions = parseImageSize(size);
          payload.width = dimensions.width;
          payload.height = dimensions.height;
        } else {
          payload.size = size;
        }
        if (selectedModelProfile.supportsQuality && quality !== 'auto') {
          payload.quality = quality;
        }
        if (group) {
          payload.group = group;
        }
        res = await API.post('/pg/images/generations', payload, {
          timeout: IMAGE_REQUEST_TIMEOUT_MS,
          skipErrorHandler: true,
        });
      }
      const data = res.data || {};
      const nextImages = Array.isArray(data.data) ? data.data : [];
      setRawResponse(data);
      setImages(nextImages);
      setElapsedMs(Math.round(performance.now() - startedAt));
      if (nextImages.length > 0) {
        const historyItem = {
          id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
          createdAt: new Date().toISOString(),
          model,
          group,
          prompt: trimmedPrompt,
          size,
          quality,
          mode: referenceImages.length > 0 ? 'edit' : 'generate',
          images: nextImages,
        };
        setHistory((current) => persistHistory([historyItem, ...current]));
      }
    } catch (error) {
      const message =
        error.response?.data?.error?.message ||
        error.response?.data?.message ||
        error.message ||
        '图片生成失败';
      Toast.error(message);
      setRawResponse(error.response?.data || { error: message });
      setElapsedMs(Math.round(performance.now() - startedAt));
    } finally {
      setGenerating(false);
    }
  };

  return (
    <div className='min-h-full bg-gradient-to-br from-slate-50 via-white to-cyan-50 p-4 md:p-8'>
      <div className='mx-auto flex max-w-7xl flex-col gap-5'>
        <div className='flex flex-col gap-2'>
          <div className='flex items-center gap-3'>
            <div className='flex h-11 w-11 items-center justify-center rounded-2xl bg-cyan-600 text-white shadow-lg shadow-cyan-200'>
              <ImageIcon size={22} />
            </div>
            <div>
              <Title heading={3} className='!m-0'>
                Image操场
              </Title>
              <Text type='tertiary'>
                直接调用图片生成接口，模型列表只展示图片生成模型。
              </Text>
            </div>
          </div>
        </div>

        <div className='grid grid-cols-1 gap-5 xl:grid-cols-[420px_1fr]'>
          <Card className='!rounded-2xl border-0 shadow-sm'>
            <div className='mb-5 flex items-center justify-between'>
              <div>
                <Text strong>生成参数</Text>
                <div className='mt-1 text-xs text-gray-500'>
                  支持文本生图；选择 gpt-image 系列模型并上传参考图后会走改图。
                </div>
              </div>
              {selectedModelMeta?.description ? (
                <Tag color='cyan'>{IMAGE_ENDPOINT_TYPE}</Tag>
              ) : null}
            </div>

            <div className='flex flex-col gap-4'>
              <div>
                <Text className='mb-2 block'>模型</Text>
                <Select
                  value={model}
                  onChange={setModel}
                  optionList={models}
                  loading={loadingModels}
                  placeholder='选择图片模型'
                  emptyContent='没有可用图片模型'
                  filter
                  style={{ width: '100%' }}
                />
                {canEditSelectedModel ? (
                  <div className='mt-3 rounded-2xl border-2 border-red-500 bg-red-50 px-4 py-3 text-red-700 shadow-sm shadow-red-100'>
                    <div className='mb-1 flex items-center gap-2 text-base font-extrabold'>
                      <AlertTriangle size={18} strokeWidth={2.6} />
                      GPT 生图扣费提醒
                    </div>
                    <div className='text-sm font-bold leading-6'>
                      GPT生图需要超过5分钟，请不要离开页面。离开后如果后台生图成功依然会扣费，账户扣费不退款。
                    </div>
                  </div>
                ) : null}
              </div>

              <div>
                <Text className='mb-2 block'>分组</Text>
                <Select
                  value={group}
                  onChange={setGroup}
                  optionList={groups}
                  placeholder='默认分组'
                  style={{ width: '100%' }}
                  disabled={groups.length === 0}
                />
              </div>

              <div>
                <Text className='mb-2 block'>尺寸</Text>
                <Select
                  value={size}
                  onChange={setSize}
                  optionList={selectedModelProfile.sizeOptions}
                  style={{ width: '100%' }}
                />
              </div>

              {selectedModelProfile.supportsQuality ? (
                <div>
                  <Text className='mb-2 block'>图片质量</Text>
                  <Select
                    value={quality}
                    onChange={setQuality}
                    optionList={selectedModelProfile.qualityOptions}
                    style={{ width: '100%' }}
                  />
                </div>
              ) : null}

              {selectedModelProfile.supportsN ? (
                <div>
                  <Text className='mb-2 block'>数量</Text>
                  <InputNumber
                    value={n}
                    min={1}
                    max={4}
                    step={1}
                    onChange={(value) => setN(Number(value) || 1)}
                    style={{ width: '100%' }}
                  />
                </div>
              ) : null}

              <div>
                <Text className='mb-2 block'>提示词</Text>
                <TextArea
                  value={prompt}
                  onChange={setPrompt}
                  autosize={{ minRows: 7, maxRows: 12 }}
                  placeholder='例如：A precise product render of a translucent glass cube on brushed steel, studio lighting'
                />
              </div>

              <div>
                <div className='mb-2 flex items-center justify-between'>
                  <Text>参考图片素材</Text>
                  <Text type='tertiary' size='small'>
                    最多 4 张
                  </Text>
                </div>
                <label
                  className={`flex items-center justify-center gap-2 rounded-xl border border-dashed px-4 py-3 text-sm ${
                    canEditSelectedModel
                      ? 'cursor-pointer border-cyan-300 bg-cyan-50/70 text-cyan-700 hover:bg-cyan-50'
                      : 'cursor-not-allowed border-gray-200 bg-gray-50 text-gray-400'
                  }`}
                >
                  <Upload size={16} />
                  {canEditSelectedModel
                    ? '上传参考图片'
                    : '当前模型不支持参考图改图'}
                  <input
                    type='file'
                    accept='image/*'
                    multiple
                    className='hidden'
                    disabled={!canEditSelectedModel}
                    onChange={handleReferenceUpload}
                  />
                </label>
                {!canEditSelectedModel ? (
                  <div className='mt-2 text-xs text-gray-500'>
                    请选择 gpt-image 系列模型启用参考图改图。
                  </div>
                ) : null}
                {referenceImages.length > 0 ? (
                  <div className='mt-3 grid grid-cols-2 gap-3'>
                    {referenceImages.map((item, index) => (
                      <div
                        key={`${item.name}-${index}`}
                        className='relative overflow-hidden rounded-xl border border-gray-100 bg-white'
                      >
                        <img
                          src={item.dataUrl}
                          alt={item.name}
                          className='aspect-square w-full object-cover'
                        />
                        <button
                          type='button'
                          title='删除参考图'
                          aria-label='删除参考图'
                          className='absolute right-2 top-2 rounded-full bg-black/60 p-1 text-white'
                          onClick={() => removeReferenceImage(index)}
                        >
                          <X size={14} />
                        </button>
                        <div className='truncate px-2 py-1 text-xs text-gray-500'>
                          {item.name}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : null}
              </div>

              <Button
                theme='solid'
                type='primary'
                size='large'
                icon={<Sparkles size={16} />}
                loading={generating}
                disabled={!hasModels || loadingModels}
                onClick={handleGenerate}
              >
                {isEditRequest ? '改图生成' : '生成图片'}
              </Button>
            </div>
          </Card>

          <Card className='!rounded-2xl border-0 shadow-sm'>
            <div className='mb-4 flex flex-wrap items-center justify-between gap-3'>
              <div>
                <Text strong>生成结果</Text>
                <div className='mt-1 text-xs text-gray-500'>
                  支持展示 b64_json 与 url 返回格式。
                </div>
              </div>
              {elapsedMs !== null ? (
                <Tag color='teal' prefixIcon={<Clock size={12} />}>
                  {(elapsedMs / 1000).toFixed(1)}s
                </Tag>
              ) : null}
            </div>

            <Spin spinning={generating} tip='图片生成中，请等待上游返回'>
              {images.length === 0 ? (
                <div className='flex min-h-[420px] items-center justify-center rounded-2xl border border-dashed border-gray-200 bg-white/70'>
                  <Empty
                    image={<ImageIcon size={56} className='text-gray-300' />}
                    title='还没有生成图片'
                    description={
                      loadingModels
                        ? '正在加载图片模型'
                        : hasModels
                          ? '输入提示词后点击生成'
                          : '当前用户没有可用图片模型'
                    }
                  />
                </div>
              ) : (
                <div className='grid grid-cols-1 gap-4 lg:grid-cols-2'>
                  {images.map((item, index) => {
                    const src = getImageSource(item);
                    return (
                      <div
                        key={`${src.slice(0, 40)}-${index}`}
                        className='overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-sm'
                      >
                        {src ? (
                          <img
                            src={src}
                            alt={`Generated ${index + 1}`}
                            className='aspect-square w-full object-cover'
                          />
                        ) : (
                          <div className='flex aspect-square items-center justify-center bg-gray-50 text-gray-400'>
                            无法展示该返回项
                          </div>
                        )}
                        <div className='flex items-center justify-between gap-3 p-3'>
                          <Text type='tertiary' size='small'>
                            #{index + 1}
                          </Text>
                          <Button
                            size='small'
                            icon={<Download size={14} />}
                            disabled={!src}
                            onClick={() => downloadImage(src, index)}
                          >
                            下载
                          </Button>
                        </div>
                        {item.revised_prompt ? (
                          <div className='border-t border-gray-100 p-3 text-xs text-gray-500'>
                            {item.revised_prompt}
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              )}
            </Spin>

            {rawResponse ? (
              <details className='mt-5 rounded-2xl bg-slate-950 p-4 text-xs text-slate-100'>
                <summary className='cursor-pointer text-slate-300'>
                  查看原始响应
                </summary>
                <pre className='mt-3 max-h-72 overflow-auto whitespace-pre-wrap break-all'>
                  {JSON.stringify(rawResponse, null, 2)}
                </pre>
              </details>
            ) : null}

            <div className='mt-6 rounded-2xl border border-gray-100 bg-white/80 p-4'>
              <div className='mb-3 flex items-center justify-between gap-3'>
                <div>
                  <Text strong>本地生成历史</Text>
                  <div className='mt-1 text-xs text-gray-500'>
                    仅缓存在当前浏览器，最多保留 {MAX_HISTORY_ITEMS} 条。
                  </div>
                </div>
                <Button
                  size='small'
                  type='danger'
                  theme='borderless'
                  icon={<Trash2 size={14} />}
                  disabled={history.length === 0}
                  onClick={clearHistory}
                >
                  清空
                </Button>
              </div>

              {history.length === 0 ? (
                <div className='rounded-xl bg-gray-50 px-4 py-6 text-center text-sm text-gray-500'>
                  暂无本地缓存记录
                </div>
              ) : (
                <div className='flex flex-col gap-3'>
                  {history.map((item) => (
                    <div
                      key={item.id}
                      className='rounded-xl border border-gray-100 bg-white p-3'
                    >
                      <div className='mb-2 flex flex-wrap items-center justify-between gap-2'>
                        <div className='flex flex-wrap items-center gap-2'>
                          <Tag color='cyan'>{item.model}</Tag>
                          <Tag>{item.size}</Tag>
                          {item.quality && item.quality !== 'auto' ? (
                            <Tag>{item.quality}</Tag>
                          ) : null}
                        </div>
                        <Text type='tertiary' size='small'>
                          {new Date(item.createdAt).toLocaleString()}
                        </Text>
                      </div>
                      <div className='mb-3 line-clamp-2 text-sm text-gray-600'>
                        {item.prompt}
                      </div>
                      <div className='grid grid-cols-3 gap-2 md:grid-cols-4'>
                        {(item.images || []).map((image, index) => {
                          const src = getImageSource(image);
                          return (
                            <button
                              key={`${item.id}-${index}`}
                              type='button'
                              className='overflow-hidden rounded-lg border border-gray-100 bg-gray-50'
                              onClick={() => src && downloadImage(src, index)}
                              title='点击下载'
                            >
                              {src ? (
                                <img
                                  src={src}
                                  alt={`History ${index + 1}`}
                                  className='aspect-square w-full object-cover'
                                />
                              ) : (
                                <div className='flex aspect-square items-center justify-center text-xs text-gray-400'>
                                  无图
                                </div>
                              )}
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default ImagePlayground;
