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
  Input,
  InputNumber,
  Select,
  Spin,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { Image as ImageIcon, Sparkles, Download, Clock } from 'lucide-react';
import { API } from '../../helpers';

const { Text, Title } = Typography;
const IMAGE_ENDPOINT_TYPE = 'image-generation';

const sizeOptions = [
  { label: '1024x1024', value: '1024x1024' },
  { label: '1024x1536', value: '1024x1536' },
  { label: '1536x1024', value: '1536x1024' },
  { label: '1024x1792', value: '1024x1792' },
  { label: '1792x1024', value: '1792x1024' },
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
    }))
    .sort((a, b) => a.value.localeCompare(b.value));
};

const buildFallbackModelOptions = (models) => {
  if (!Array.isArray(models)) {
    return [];
  }
  return models
    .filter(isImageModelName)
    .map((model) => ({ label: model, value: model }))
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

const ImagePlayground = () => {
  const [models, setModels] = useState([]);
  const [groups, setGroups] = useState([]);
  const [model, setModel] = useState('');
  const [group, setGroup] = useState('');
  const [prompt, setPrompt] = useState('');
  const [size, setSize] = useState('1024x1024');
  const [n, setN] = useState(1);
  const [loadingModels, setLoadingModels] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [elapsedMs, setElapsedMs] = useState(null);
  const [images, setImages] = useState([]);
  const [rawResponse, setRawResponse] = useState(null);

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

    const payload = {
      model,
      prompt: trimmedPrompt,
      n,
      size,
    };
    if (group) {
      payload.group = group;
    }

    const startedAt = performance.now();
    setGenerating(true);
    setElapsedMs(null);
    setImages([]);
    setRawResponse(null);

    try {
      const res = await API.post('/pg/images/generations', payload, {
        timeout: 190000,
        skipErrorHandler: true,
      });
      const data = res.data || {};
      setRawResponse(data);
      setImages(Array.isArray(data.data) ? data.data : []);
      setElapsedMs(Math.round(performance.now() - startedAt));
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
                  通过 /pg/images/generations 走当前登录用户权限与计费。
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
                  optionList={sizeOptions}
                  style={{ width: '100%' }}
                />
              </div>

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

              <div>
                <Text className='mb-2 block'>提示词</Text>
                <Input.TextArea
                  value={prompt}
                  onChange={setPrompt}
                  autosize={{ minRows: 7, maxRows: 12 }}
                  placeholder='例如：A precise product render of a translucent glass cube on brushed steel, studio lighting'
                />
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
                生成图片
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
          </Card>
        </div>
      </div>
    </div>
  );
};

export default ImagePlayground;
