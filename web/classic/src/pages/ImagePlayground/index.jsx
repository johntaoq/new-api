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
  Modal,
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
  AtSign,
  ZoomIn,
  ZoomOut,
  RotateCcw,
} from 'lucide-react';
import { API } from '../../helpers';

const { Text, Title } = Typography;
const IMAGE_ENDPOINT_TYPE = 'image-generation';
const IMAGE_HISTORY_STORAGE_KEY = 'image_playground_history';
const MAX_HISTORY_ITEMS = 8;
const MAX_HISTORY_CHARS = 4000000;
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

const defaultSizeOptions = [{ label: '1024x1024', value: '1024x1024' }];

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

const isMaiReferenceImageFile = (file) => {
  const type = (file?.type || '').toLowerCase();
  const name = file?.name || '';
  return (
    type === 'image/png' ||
    type === 'image/jpeg' ||
    /\.(png|jpe?g)$/i.test(name)
  );
};

const isMaiImage25Model = (modelName = '') =>
  modelName.toLowerCase().includes('mai-image-2.5');

const isEditCapableModel = (modelName = '') => {
  const normalized = modelName.toLowerCase();
  return normalized.includes('gpt-image') || isMaiImage25Model(normalized);
};

const getImageModelProfile = (modelName = '') => {
  const normalized = modelName.toLowerCase();
  if (normalized.includes('mai-image')) {
    const supportsImageEdit = isMaiImage25Model(modelName);
    return {
      kind: 'mai',
      sizeOptions: maiImageSizeOptions,
      qualityOptions: [],
      supportsQuality: false,
      supportsN: false,
      requestShape: 'width-height',
      maxReferenceImages: supportsImageEdit ? 1 : 0,
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
      maxReferenceImages: 16,
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
      maxReferenceImages: 16,
    };
  }
  return {
    kind: 'default',
    sizeOptions: defaultSizeOptions,
    qualityOptions: [],
    supportsQuality: false,
    supportsN: true,
    requestShape: 'size',
    maxReferenceImages: 0,
  };
};

const parseImageSize = (value = '1024x1024') => {
  const [width, height] = value.split('x').map((item) => Number(item));
  return {
    width: Number.isFinite(width) ? width : 1024,
    height: Number.isFinite(height) ? height : 1024,
  };
};

const buildImageModelOptions = (models, usableGroup) => {
  if (!Array.isArray(models)) {
    return [];
  }

  const usableGroups = Object.keys(usableGroup || {}).filter(
    (groupName) => groupName && groupName !== 'auto',
  );
  const usableGroupSet = new Set(usableGroups);

  return models
    .filter((model) => {
      const supportedEndpoints = model.supported_endpoint_types || [];
      const supportsImageEndpoint =
        Array.isArray(supportedEndpoints) &&
        supportedEndpoints.includes(IMAGE_ENDPOINT_TYPE);
      return supportsImageEndpoint || isImageModelName(model.model_name);
    })
    .filter((model) => {
      const enabledGroups = Array.isArray(model.enable_groups)
        ? model.enable_groups
        : [];
      if (enabledGroups.includes('all')) {
        return usableGroups.length > 0;
      }
      return enabledGroups.some((item) => usableGroupSet.has(item));
    })
    .map((model) => {
      const enabledGroups = Array.isArray(model.enable_groups)
        ? model.enable_groups
        : [];
      const availableGroups = enabledGroups.includes('all')
        ? usableGroups
        : enabledGroups.filter((item) => usableGroupSet.has(item));
      return {
        label: model.model_name,
        value: model.model_name,
        description: model.supported_endpoint_types?.join(', ') || '',
        editCapable: isEditCapableModel(model.model_name),
        enableGroups: availableGroups,
      };
    })
    .sort((a, b) => a.value.localeCompare(b.value));
};

const buildImageGroupOptions = (imageModels, usableGroup, autoGroups) => {
  const groupsWithImageModels = new Set(
    imageModels.flatMap((item) => item.enableGroups || []),
  );
  const options = Object.entries(usableGroup || {})
    .filter(([value]) => value && value !== 'auto')
    .filter(([value]) => groupsWithImageModels.has(value))
    .map(([value, description]) => ({
      label: description || value,
      value,
    }));

  const autoImageGroups = (Array.isArray(autoGroups) ? autoGroups : []).filter(
    (value) => groupsWithImageModels.has(value),
  );
  if (usableGroup?.auto && autoImageGroups.length > 0) {
    options.unshift({
      label: usableGroup.auto || '自动选择',
      value: 'auto',
    });
  }
  return options;
};

const modelSupportsImageGroup = (modelOption, group, autoGroups) => {
  const enabledGroups = modelOption?.enableGroups || [];
  if (group === 'auto') {
    return (Array.isArray(autoGroups) ? autoGroups : []).some((item) =>
      enabledGroups.includes(item),
    );
  }
  return enabledGroups.includes(group);
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

const normalizeReferenceFileName = (
  name = 'referenced-image',
  contentType = 'image/png',
) => {
  const cleanName =
    String(name)
      .replace(/\.[a-z0-9]+$/i, '')
      .replace(/[^a-zA-Z0-9._-]+/g, '-')
      .replace(/^-+|-+$/g, '') || 'referenced-image';
  const extension =
    contentType === 'image/jpeg'
      ? 'jpg'
      : contentType === 'image/webp'
        ? 'webp'
        : contentType === 'image/gif'
          ? 'gif'
          : 'png';
  return `${cleanName}.${extension}`;
};

const imageSourceToReferenceImage = async (src, name) => {
  let blob;
  if (src.startsWith('data:') || src.startsWith('blob:')) {
    const response = await fetch(src);
    if (!response.ok) {
      throw new Error('Failed to load image source');
    }
    blob = await response.blob();
  } else {
    const response = await API.post(
      '/pg/images/reference',
      { url: src },
      {
        responseType: 'blob',
        timeout: 60000,
        skipErrorHandler: true,
      },
    );
    blob = response.data;
  }
  if (!blob || !blob.type.startsWith('image/')) {
    throw new Error('Reference source is not an image');
  }
  const fileName = normalizeReferenceFileName(name, blob.type);
  const file = new File([blob], fileName, {
    type: blob.type || 'image/png',
  });
  return {
    name: fileName,
    size: file.size,
    file,
    dataUrl: await fileToDataUrl(file),
  };
};

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
  while (next.length > 0 && JSON.stringify(next).length > MAX_HISTORY_CHARS) {
    next = next.slice(0, -1);
  }
  localStorage.setItem(IMAGE_HISTORY_STORAGE_KEY, JSON.stringify(next));
  return next;
};

const ImagePlayground = () => {
  const [allImageModels, setAllImageModels] = useState([]);
  const [autoGroups, setAutoGroups] = useState([]);
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
  const [preview, setPreview] = useState({
    visible: false,
    src: '',
    title: '',
    scale: 1,
    width: 0,
    height: 0,
  });

  const models = useMemo(
    () =>
      allImageModels.filter((item) =>
        modelSupportsImageGroup(item, group, autoGroups),
      ),
    [allImageModels, autoGroups, group],
  );
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
        );
        const nextAutoGroups = Array.isArray(pricingRes.data.auto_groups)
          ? pricingRes.data.auto_groups
          : [];
        const groupOptions = buildImageGroupOptions(
          options,
          pricingRes.data.usable_group,
          nextAutoGroups,
        );
        setAllImageModels(options);
        setAutoGroups(nextAutoGroups);
        setGroups(groupOptions);
        setGroup((current) =>
          groupOptions.some((option) => option.value === current)
            ? current
            : groupOptions[0]?.value || '',
        );
        return;
      }
      throw new Error(pricingRes.data?.message || '加载图片模型与分组失败');
    } catch (error) {
      Toast.error('加载图片模型失败');
      setAllImageModels([]);
      setAutoGroups([]);
      setGroups([]);
      setGroup('');
      setModel('');
    } finally {
      setLoadingModels(false);
    }
  }, []);

  useEffect(() => {
    loadModels();
  }, [loadModels]);

  useEffect(() => {
    setModel((current) =>
      models.some((option) => option.value === current)
        ? current
        : models[0]?.value || '',
    );
  }, [models]);

  const selectedModelMeta = useMemo(
    () => models.find((item) => item.value === model),
    [model, models],
  );
  const selectedModelProfile = useMemo(
    () => getImageModelProfile(model),
    [model],
  );
  const maxReferenceImages = selectedModelProfile.maxReferenceImages || 0;
  const canEditSelectedModel =
    maxReferenceImages > 0 && isEditCapableModel(model);
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
      return;
    }
    if (canEditSelectedModel && referenceImages.length > maxReferenceImages) {
      setReferenceImages((current) => current.slice(0, maxReferenceImages));
    }
  }, [canEditSelectedModel, maxReferenceImages, referenceImages.length]);

  const handleReferenceUpload = async (event) => {
    const files = Array.from(event.target.files || []);
    event.target.value = '';
    if (files.length === 0) {
      return;
    }
    if (!canEditSelectedModel || maxReferenceImages <= 0) {
      Toast.warning('当前模型不支持参考图改图');
      return;
    }

    let imageFiles = files.filter((file) => file.type.startsWith('image/'));
    if (imageFiles.length !== files.length) {
      Toast.warning('只能上传图片文件');
    }
    if (selectedModelProfile.kind === 'mai') {
      const beforeFilterCount = imageFiles.length;
      imageFiles = imageFiles.filter(isMaiReferenceImageFile);
      if (imageFiles.length !== beforeFilterCount) {
        Toast.warning('MAI-Image-2.5 编辑仅支持 PNG 或 JPEG 参考图');
      }
    }
    const remainingSlots = maxReferenceImages - referenceImages.length;
    if (remainingSlots <= 0) {
      Toast.warning(`参考图最多 ${maxReferenceImages} 张`);
      return;
    }
    if (imageFiles.length > remainingSlots) {
      Toast.warning(`已按模型限制保留前 ${remainingSlots} 张参考图`);
    }

    try {
      const loaded = await Promise.all(
        imageFiles.slice(0, remainingSlots).map(async (file) => ({
          name: file.name,
          size: file.size,
          file,
          dataUrl: await fileToDataUrl(file),
        })),
      );
      setReferenceImages((current) =>
        [...current, ...loaded].slice(0, maxReferenceImages),
      );
    } catch (error) {
      Toast.error('读取参考图片失败');
    }
  };

  const addReferenceImageFromSource = async (src, name) => {
    if (!src) {
      return;
    }
    if (!canEditSelectedModel || maxReferenceImages <= 0) {
      Toast.warning(
        '当前模型不支持参考图改图，请切换到 gpt-image 或 MAI-Image-2.5 系列模型',
      );
      return;
    }
    if (referenceImages.length >= maxReferenceImages) {
      Toast.warning(`参考图最多 ${maxReferenceImages} 张`);
      return;
    }

    try {
      const referenceImage = await imageSourceToReferenceImage(src, name);
      if (
        selectedModelProfile.kind === 'mai' &&
        !isMaiReferenceImageFile(referenceImage.file)
      ) {
        Toast.warning('MAI-Image-2.5 编辑仅支持 PNG 或 JPEG 参考图');
        return;
      }
      setReferenceImages((current) => {
        if (current.length >= maxReferenceImages) {
          return current;
        }
        return [...current, referenceImage].slice(0, maxReferenceImages);
      });
      Toast.success('已加入参考图');
    } catch (error) {
      Toast.error('无法引用该图片，请下载后手动上传');
    }
  };

  const removeReferenceImage = (index) => {
    setReferenceImages((current) => current.filter((_, i) => i !== index));
  };

  const clearHistory = () => {
    localStorage.removeItem(IMAGE_HISTORY_STORAGE_KEY);
    setHistory([]);
  };

  const deleteHistoryImage = (historyId, imageIndex) => {
    setHistory((current) => {
      const next = current.reduce((items, item) => {
        if (item.id !== historyId) {
          items.push(item);
          return items;
        }

        const nextImages = (item.images || []).filter(
          (_, index) => index !== imageIndex,
        );
        if (nextImages.length > 0) {
          items.push({ ...item, images: nextImages });
        }
        return items;
      }, []);
      return persistHistory(next);
    });
  };

  const openPreview = (src, title) => {
    if (!src) {
      return;
    }
    setPreview({
      visible: true,
      src,
      title,
      scale: 1,
      width: 0,
      height: 0,
    });
  };

  const closePreview = () => {
    setPreview((current) => ({ ...current, visible: false }));
  };

  const updatePreviewScale = (delta) => {
    setPreview((current) => ({
      ...current,
      scale: Math.min(
        4,
        Math.max(0.25, Number((current.scale + delta).toFixed(2))),
      ),
    }));
  };

  const resetPreviewScale = () => {
    setPreview((current) => ({ ...current, scale: 1 }));
  };

  const updatePreviewSize = (event) => {
    const { naturalWidth, naturalHeight } = event.currentTarget;
    setPreview((current) => ({
      ...current,
      width: naturalWidth,
      height: naturalHeight,
    }));
  };

  const getPreviewImageStyle = () => {
    if (!preview.width || !preview.height) {
      return {
        maxWidth: '100%',
        maxHeight: '72vh',
        width: 'auto',
        height: 'auto',
      };
    }

    const viewportWidth = Math.max(
      280,
      Math.min(window.innerWidth * 0.96, 1180) - 40,
    );
    const viewportHeight = Math.max(240, window.innerHeight * 0.72);
    const fitRatio = Math.min(
      viewportWidth / preview.width,
      viewportHeight / preview.height,
      1,
    );

    return {
      width: `${preview.width * fitRatio * preview.scale}px`,
      height: `${preview.height * fitRatio * preview.scale}px`,
      maxWidth: 'none',
      maxHeight: 'none',
      transition: 'width 120ms ease-out, height 120ms ease-out',
    };
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
      Toast.warning(
        '当前模型不支持参考图改图，请选择 gpt-image 或 MAI-Image-2.5 系列模型',
      );
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
        if (selectedModelProfile.kind !== 'mai') {
          formData.append('n', String(n));
          formData.append('size', size);
          if (selectedModelProfile.supportsQuality && quality !== 'auto') {
            formData.append('quality', quality);
          }
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
    <div className='min-h-full bg-gradient-to-br from-slate-50 via-white to-cyan-50 px-4 pb-4 pt-20 md:px-8 md:pb-8 md:pt-24'>
      <div className='mx-auto flex max-w-7xl flex-col gap-5'>
        <div className='flex flex-col gap-2'>
          <div className='flex items-center gap-3'>
            <div className='flex h-11 w-11 items-center justify-center rounded-2xl bg-cyan-600 text-white shadow-lg shadow-cyan-200'>
              <ImageIcon size={22} />
            </div>
            <div>
              <Title heading={3} className='!m-0'>
                图片生成
              </Title>
            </div>
          </div>
        </div>

        <div className='grid grid-cols-1 gap-5 xl:grid-cols-[420px_1fr]'>
          <Card className='!rounded-2xl border-0 shadow-sm'>
            <div className='mb-5 flex items-center justify-between'>
              <div>
                <Text strong>生成参数</Text>
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
                      生图扣费提醒
                    </div>
                    <div className='text-sm font-bold leading-6'>
                      生图可能需要超过5分钟，请不要离开页面。离开后如果后台生图成功依然会扣费，账户扣费不退款。
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
                  placeholder='选择图片模型分组'
                  emptyContent='当前权限下没有包含图片模型的分组'
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
                    {canEditSelectedModel
                      ? `最多 ${maxReferenceImages} 张`
                      : '当前模型不支持'}
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
                    accept={
                      selectedModelProfile.kind === 'mai'
                        ? 'image/png,image/jpeg'
                        : 'image/*'
                    }
                    multiple
                    className='hidden'
                    disabled={!canEditSelectedModel}
                    onChange={handleReferenceUpload}
                  />
                </label>
                {!canEditSelectedModel ? (
                  <div className='mt-2 text-xs text-gray-500'>
                    请选择 gpt-image 或 MAI-Image-2.5 系列模型启用参考图改图。
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
                          <button
                            type='button'
                            className='group relative block w-full cursor-zoom-in overflow-hidden'
                            onClick={() =>
                              openPreview(src, `生成图片 #${index + 1}`)
                            }
                            title='点击预览'
                          >
                            <img
                              src={src}
                              alt={`Generated ${index + 1}`}
                              className='aspect-square w-full object-cover transition-transform duration-200 group-hover:scale-[1.02]'
                            />
                            <span className='absolute inset-x-0 bottom-0 bg-black/55 px-3 py-2 text-left text-xs font-medium text-white opacity-0 transition-opacity group-hover:opacity-100'>
                              点击预览
                            </span>
                          </button>
                        ) : (
                          <div className='flex aspect-square items-center justify-center bg-gray-50 text-gray-400'>
                            无法展示该返回项
                          </div>
                        )}
                        <div className='flex items-center justify-between gap-3 p-3'>
                          <Text type='tertiary' size='small'>
                            #{index + 1}
                          </Text>
                          <div className='flex items-center gap-2'>
                            <Button
                              size='small'
                              icon={<AtSign size={14} />}
                              disabled={!src || !canEditSelectedModel}
                              onClick={() =>
                                addReferenceImageFromSource(
                                  src,
                                  `generated-${index + 1}`,
                                )
                              }
                            >
                              引用
                            </Button>
                            <Button
                              size='small'
                              icon={<Download size={14} />}
                              disabled={!src}
                              onClick={() => downloadImage(src, index)}
                            >
                              下载
                            </Button>
                          </div>
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
                      <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
                        {(item.images || []).map((image, index) => {
                          const src = getImageSource(image);
                          return (
                            <div
                              key={`${item.id}-${index}`}
                              className='group relative overflow-hidden rounded-lg border border-gray-100 bg-gray-50'
                            >
                              {src ? (
                                <>
                                  <button
                                    type='button'
                                    className='block w-full cursor-zoom-in'
                                    onClick={() =>
                                      openPreview(src, `历史图片 #${index + 1}`)
                                    }
                                    title='点击预览'
                                  >
                                    <img
                                      src={src}
                                      alt={`History ${index + 1}`}
                                      className='aspect-square w-full object-cover transition-transform duration-200 group-hover:scale-[1.04]'
                                    />
                                  </button>
                                  <div className='absolute inset-x-1 bottom-1 flex items-center justify-between gap-1 opacity-0 transition-opacity group-hover:opacity-100'>
                                    <button
                                      type='button'
                                      title='引用为改图参考'
                                      aria-label='引用为改图参考'
                                      className='rounded-full bg-cyan-600/90 p-1.5 text-white shadow-sm disabled:cursor-not-allowed disabled:bg-gray-500/70'
                                      disabled={!canEditSelectedModel}
                                      onClick={() =>
                                        addReferenceImageFromSource(
                                          src,
                                          `history-${index + 1}`,
                                        )
                                      }
                                    >
                                      <AtSign size={13} />
                                    </button>
                                    <button
                                      type='button'
                                      title='删除这张历史图片'
                                      aria-label='删除这张历史图片'
                                      className='rounded-full bg-black/70 p-1.5 text-white shadow-sm hover:bg-red-600'
                                      onClick={() =>
                                        deleteHistoryImage(item.id, index)
                                      }
                                    >
                                      <Trash2 size={13} />
                                    </button>
                                  </div>
                                </>
                              ) : (
                                <div className='flex aspect-square items-center justify-center text-xs text-gray-400'>
                                  无图
                                </div>
                              )}
                            </div>
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
      <Modal
        visible={preview.visible}
        title={preview.title || '图片预览'}
        onCancel={closePreview}
        footer={null}
        width='min(96vw, 1180px)'
        bodyStyle={{ padding: 0 }}
        keepDOM={false}
      >
        <div className='flex max-h-[82vh] flex-col bg-slate-950'>
          <div className='flex flex-wrap items-center justify-between gap-2 border-b border-white/10 px-4 py-3 text-white'>
            <div className='text-sm font-semibold'>
              缩放 {(preview.scale * 100).toFixed(0)}%
            </div>
            <div className='flex flex-wrap items-center gap-2'>
              <Button
                size='small'
                icon={<ZoomOut size={14} />}
                onClick={() => updatePreviewScale(-0.25)}
              >
                缩小
              </Button>
              <Button
                size='small'
                icon={<RotateCcw size={14} />}
                onClick={resetPreviewScale}
              >
                重置
              </Button>
              <Button
                size='small'
                icon={<ZoomIn size={14} />}
                onClick={() => updatePreviewScale(0.25)}
              >
                放大
              </Button>
              <Button
                size='small'
                icon={<AtSign size={14} />}
                disabled={!preview.src || !canEditSelectedModel}
                onClick={() =>
                  addReferenceImageFromSource(
                    preview.src,
                    preview.title || 'preview-image',
                  )
                }
              >
                引用
              </Button>
              <Button
                size='small'
                type='primary'
                icon={<Download size={14} />}
                onClick={() => downloadImage(preview.src, 0)}
              >
                下载
              </Button>
            </div>
          </div>
          <div className='flex-1 overflow-auto p-5'>
            {preview.src ? (
              <div className='flex min-h-[58vh] min-w-full items-center justify-center'>
                <img
                  src={preview.src}
                  alt={preview.title || '图片预览'}
                  onLoad={updatePreviewSize}
                  className='max-w-none select-none rounded-lg bg-white shadow-2xl'
                  style={getPreviewImageStyle()}
                />
              </div>
            ) : null}
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default ImagePlayground;
