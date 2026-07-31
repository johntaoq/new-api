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
import { Link } from 'react-router-dom';
import { Typography, Tag } from '@douyinfe/semi-ui';
import SkeletonWrapper from '../components/SkeletonWrapper';

const HeaderLogo = ({
  isMobile,
  isConsoleRoute,
  isLoading,
  systemName,
  isHomeRoute,
  isSelfUseMode,
  isDemoSiteMode,
  t,
}) => {
  if (isMobile && isConsoleRoute) {
    return null;
  }

  const displayLogo = '/unikeyx-logo-header.png';
  const logoBoxClassName = isHomeRoute
    ? 'ukx-home-logo-box relative h-9 w-[68px] md:h-10 md:w-[75.56px] shrink-0 overflow-hidden rounded-lg bg-white shadow-sm ring-1 ring-white/20'
    : 'relative h-9 w-[68px] md:h-10 md:w-[75.56px] shrink-0 overflow-hidden rounded-lg bg-white shadow-sm ring-1 ring-black/5 dark:ring-white/10';
  const logoImageClassName =
    'absolute inset-0 h-full w-full object-contain opacity-100 transition-transform duration-200 group-hover:scale-[1.02]';
  const showLogoSkeleton = false;
  const showTitleSkeleton = !isHomeRoute && isLoading;

  return (
    <Link to='/' className='group flex items-center gap-2'>
      <div className={logoBoxClassName}>
        <SkeletonWrapper loading={showLogoSkeleton} type='image' />
        <img
          src={displayLogo}
          alt='logo'
          className={logoImageClassName}
        />
      </div>
      <div className='hidden md:flex items-center gap-2'>
        <div className='flex items-center gap-2'>
          <SkeletonWrapper
            loading={showTitleSkeleton}
            type='title'
            width={120}
            height={24}
          >
            <Typography.Title
              heading={4}
              className={`!text-lg !font-semibold !mb-0 ${isHomeRoute ? 'ukx-home-logo-text !text-white' : ''}`}
            >
              {systemName}
            </Typography.Title>
          </SkeletonWrapper>
          {(isSelfUseMode || isDemoSiteMode) && !isLoading && (
            <Tag
              color={isSelfUseMode ? 'purple' : 'blue'}
              className='text-xs px-1.5 py-0.5 rounded whitespace-nowrap shadow-sm'
              size='small'
              shape='circle'
            >
              {isSelfUseMode ? t('自用模式') : t('演示站点')}
            </Tag>
          )}
        </div>
      </div>
    </Link>
  );
};

export default HeaderLogo;
