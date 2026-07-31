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

import React, { lazy, Suspense } from 'react';
import { useLocation } from 'react-router-dom';
import { useHeaderBar } from '../../../hooks/common/useHeaderBar';
import { useNotifications } from '../../../hooks/common/useNotifications';
import { useNavigation } from '../../../hooks/common/useNavigation';
import MobileMenuButton from './MobileMenuButton';
import HeaderLogo from './HeaderLogo';
import Navigation from './Navigation';
import ActionButtons from './ActionButtons';

const NoticeModal = lazy(() => import('../NoticeModal'));

const HeaderBar = ({ onMobileMenuToggle, drawerOpen }) => {
  const location = useLocation();
  const isHomeRoute = location.pathname === '/';
  const {
    userState,
    statusState,
    isMobile,
    collapsed,
    logoLoaded,
    currentLang,
    isLoading,
    systemName,
    logo,
    isNewYear,
    isSelfUseMode,
    docsLink,
    isDemoSiteMode,
    isConsoleRoute,
    theme,
    headerNavModules,
    pricingRequireAuth,
    logout,
    handleLanguageChange,
    handleThemeToggle,
    handleMobileMenuToggle,
    navigate,
    t,
  } = useHeaderBar({ onMobileMenuToggle, drawerOpen });

  const {
    noticeVisible,
    unreadCount,
    handleNoticeOpen,
    handleNoticeClose,
    getUnreadKeys,
  } = useNotifications(statusState);

  const { mainNavLinks } = useNavigation(t, docsLink, headerNavModules);
  const headerIsLoading = isHomeRoute ? false : isLoading;
  const headerClassName = isHomeRoute
    ? 'ukx-home-headerbar text-white sticky top-0 z-50 transition-colors duration-300 bg-slate-950/90 backdrop-blur-xl border-b border-white/10 shadow-[0_12px_34px_rgba(2,8,23,0.26)]'
    : 'text-semi-color-text-0 sticky top-0 z-50 transition-colors duration-300 bg-white/75 dark:bg-zinc-900/75 backdrop-blur-lg';

  return (
    <header className={headerClassName}>
      {isHomeRoute && (
        <style>{`
          .ukx-home-headerbar,
          .ukx-home-headerbar a,
          .ukx-home-headerbar span,
          .ukx-home-headerbar .semi-typography,
          .ukx-home-headerbar .semi-typography h1,
          .ukx-home-headerbar .semi-typography h2,
          .ukx-home-headerbar .semi-typography h3,
          .ukx-home-headerbar .semi-typography h4,
          .ukx-home-headerbar .semi-typography h5,
          .ukx-home-headerbar .semi-typography h6 {
            color: rgba(255, 255, 255, 0.94) !important;
          }

          .ukx-home-headerbar a:hover {
            color: #fff !important;
            background: rgba(255, 255, 255, 0.1);
          }

          .ukx-home-headerbar button:not(.semi-button-primary) {
            color: rgba(255, 255, 255, 0.92) !important;
            background: rgba(255, 255, 255, 0.11) !important;
            border-color: rgba(255, 255, 255, 0.14) !important;
          }

          .ukx-home-headerbar button:not(.semi-button-primary):hover {
            background: rgba(255, 255, 255, 0.18) !important;
          }

          .ukx-home-headerbar .ukx-home-logo-box {
            background: #fff !important;
          }

          .ukx-home-headerbar .ukx-home-logo-text {
            color: #fff !important;
            text-shadow: 0 1px 16px rgba(0, 0, 0, 0.26);
          }
        `}</style>
      )}
      {noticeVisible && (
        <Suspense fallback={null}>
          <NoticeModal
            visible={noticeVisible}
            onClose={handleNoticeClose}
            isMobile={isMobile}
            defaultTab={unreadCount > 0 ? 'system' : 'inApp'}
            unreadKeys={getUnreadKeys()}
          />
        </Suspense>
      )}

      <div className='w-full px-2'>
        <div className='flex items-center justify-between h-16'>
          <div className='flex items-center'>
            <MobileMenuButton
              isConsoleRoute={isConsoleRoute}
              isMobile={isMobile}
              drawerOpen={drawerOpen}
              collapsed={collapsed}
              onToggle={handleMobileMenuToggle}
              t={t}
            />

            <HeaderLogo
              isMobile={isMobile}
              isConsoleRoute={isConsoleRoute}
              logo={logo}
              logoLoaded={logoLoaded}
              isLoading={headerIsLoading}
              systemName={systemName}
              isHomeRoute={isHomeRoute}
              isSelfUseMode={isSelfUseMode}
              isDemoSiteMode={isDemoSiteMode}
              t={t}
            />
          </div>

          <Navigation
            mainNavLinks={mainNavLinks}
            isMobile={isMobile}
            isLoading={headerIsLoading}
            userState={userState}
            pricingRequireAuth={pricingRequireAuth}
            isHomeRoute={isHomeRoute}
          />

          <ActionButtons
            isNewYear={isNewYear}
            unreadCount={unreadCount}
            onNoticeOpen={handleNoticeOpen}
            theme={theme}
            onThemeToggle={handleThemeToggle}
            currentLang={currentLang}
            onLanguageChange={handleLanguageChange}
            userState={userState}
            isLoading={headerIsLoading}
            isMobile={isMobile}
            isSelfUseMode={isSelfUseMode}
            logout={logout}
            navigate={navigate}
            t={t}
          />
        </div>
      </div>
    </header>
  );
};

export default HeaderBar;
