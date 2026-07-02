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

import React from 'react';
import { Card, Avatar, Skeleton, Tag } from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

const StatsCards = ({
  groupedStatsData,
  loading,
  getTrendSpec,
  CARD_PROPS,
  CHART_CONFIG,
}) => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  return (
    <div className='mb-4'>
      <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4'>
        {groupedStatsData.map((group, idx) => (
          <Card
            key={idx}
            {...CARD_PROPS}
            className={`${group.color} border-0 !rounded-2xl w-full`}
            title={group.title}
          >
            <div className='space-y-4'>
              {group.items.map((item, itemIdx) => {
                const hasDetails = item.details?.length > 0;

                return (
                  <div
                    key={itemIdx}
                    className='cursor-pointer rounded-xl'
                    onClick={item.onClick}
                  >
                    <div
                      className={`flex justify-between gap-3 ${hasDetails ? 'items-start' : 'items-center'}`}
                    >
                      <div className='flex flex-1 min-w-0 items-start'>
                        <Avatar
                          className='mr-3 shrink-0'
                          size='small'
                          color={item.avatarColor}
                        >
                          {item.icon}
                        </Avatar>
                        <div className='min-w-0 flex-1'>
                          <div className='text-xs text-gray-500'>
                            {item.title}
                          </div>
                          <div className='text-lg font-semibold leading-tight break-all'>
                            <Skeleton
                              loading={loading}
                              active
                              placeholder={
                                <Skeleton.Paragraph
                                  active
                                  rows={1}
                                  style={{
                                    width: '65px',
                                    height: '24px',
                                    marginTop: '4px',
                                  }}
                                />
                              }
                            >
                              {item.value}
                            </Skeleton>
                          </div>
                          {hasDetails && (
                            <div className='mt-2 space-y-1 text-sm text-gray-500'>
                              {item.details.map((detail) => (
                                <div
                                  key={detail.label}
                                  className='flex items-center justify-between gap-3'
                                >
                                  <span className='shrink-0'>
                                    {detail.label}
                                  </span>
                                  <span className='min-w-0 text-right font-medium text-gray-700 break-all'>
                                    {detail.value}
                                  </span>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                      <div className='shrink-0 self-start'>
                        {item.title === t('当前余额') ? (
                          <Tag
                            color='white'
                            shape='circle'
                            size='default'
                            onClick={(e) => {
                              e.stopPropagation();
                              navigate('/console/topup');
                            }}
                          >
                            {t('充值')}
                          </Tag>
                        ) : (
                          (loading ||
                            (item.trendData && item.trendData.length > 0)) && (
                            <div className='w-24 h-10'>
                              <VChart
                                spec={getTrendSpec(
                                  item.trendData,
                                  item.trendColor,
                                )}
                                option={CHART_CONFIG}
                              />
                            </div>
                          )
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
};

export default StatsCards;
