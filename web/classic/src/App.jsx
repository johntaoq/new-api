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

import React, { lazy, Suspense, useContext, useMemo } from 'react';
import { Route, Routes, useLocation, useParams } from 'react-router-dom';
import Loading from './components/common/ui/Loading';
import { AuthRedirect, PermissionRoute, PrivateRoute } from './helpers/auth';
import { StatusContext } from './context/Status';
import SetupCheck from './components/layout/SetupCheck';

const CHUNK_LOAD_RELOAD_KEY = 'new-api-chunk-load-reloaded';

const isChunkLoadError = (error) => {
  const message = `${error?.name || ''} ${error?.message || ''}`;
  return /ChunkLoadError|Loading chunk \d+ failed|Loading CSS chunk \d+ failed|Failed to fetch dynamically imported module|Importing a module script failed|error loading dynamically imported module/i.test(
    message,
  );
};

const getChunkReloaded = () => {
  try {
    return window.sessionStorage.getItem(CHUNK_LOAD_RELOAD_KEY);
  } catch {
    return null;
  }
};

const setChunkReloaded = () => {
  try {
    window.sessionStorage.setItem(CHUNK_LOAD_RELOAD_KEY, '1');
  } catch {}
};

const clearChunkReloaded = () => {
  try {
    window.sessionStorage.removeItem(CHUNK_LOAD_RELOAD_KEY);
  } catch {}
};

const handleLazyLoadError = (error) => {
  if (typeof window === 'undefined' || !isChunkLoadError(error)) {
    throw error;
  }

  const hasReloaded = getChunkReloaded();
  if (!hasReloaded) {
    setChunkReloaded();
    window.location.reload();
    return new Promise(() => {});
  }

  clearChunkReloaded();
  throw error;
};

const lazyPage = (loader) => {
  const Component = lazy(() =>
    loader()
      .then((module) => {
        clearChunkReloaded();
        return module;
      })
      .catch(handleLazyLoadError),
  );

  return function LazyPage(props) {
    return (
      <Suspense fallback={<Loading />}>
        <Component {...props} />
      </Suspense>
    );
  };
};

const Home = lazyPage(() => import('./pages/Home'));
const Dashboard = lazyPage(() => import('./pages/Dashboard'));
const About = lazyPage(() => import('./pages/About'));
const UserAgreement = lazyPage(() => import('./pages/UserAgreement'));
const PrivacyPolicy = lazyPage(() => import('./pages/PrivacyPolicy'));
const User = lazyPage(() => import('./pages/User'));
const RegisterForm = lazyPage(() => import('./components/auth/RegisterForm'));
const LoginForm = lazyPage(() => import('./components/auth/LoginForm'));
const NotFound = lazyPage(() => import('./pages/NotFound'));
const Forbidden = lazyPage(() => import('./pages/Forbidden'));
const Setting = lazyPage(() => import('./pages/Setting'));
const PasswordResetForm = lazyPage(
  () => import('./components/auth/PasswordResetForm'),
);
const PasswordResetConfirm = lazyPage(
  () => import('./components/auth/PasswordResetConfirm'),
);
const Channel = lazyPage(() => import('./pages/Channel'));
const Token = lazyPage(() => import('./pages/Token'));
const Redemption = lazyPage(() => import('./pages/Redemption'));
const TopUp = lazyPage(() => import('./pages/TopUp'));
const Log = lazyPage(() => import('./pages/Log'));
const Chat = lazyPage(() => import('./pages/Chat'));
const Chat2Link = lazyPage(() => import('./pages/Chat2Link'));
const MjProxy = lazyPage(() => import('./pages/Midjourney'));
const Pricing = lazyPage(() => import('./pages/Pricing'));
const Task = lazyPage(() => import('./pages/Task'));
const ModelPage = lazyPage(() => import('./pages/Model'));
const ModelDeploymentPage = lazyPage(() => import('./pages/ModelDeployment'));
const Playground = lazyPage(() => import('./pages/Playground'));
const ImagePlayground = lazyPage(() => import('./pages/ImagePlayground'));
const Subscription = lazyPage(() => import('./pages/Subscription'));
const Billing = lazyPage(() => import('./pages/Billing'));
const OAuth2Callback = lazyPage(
  () => import('./components/auth/OAuth2Callback'),
);
const PersonalSetting = lazyPage(
  () => import('./components/settings/PersonalSetting'),
);
const Setup = lazyPage(() => import('./pages/Setup'));

function DynamicOAuth2Callback() {
  const { provider } = useParams();
  return <OAuth2Callback type={provider} />;
}

function App() {
  const location = useLocation();
  const [statusState] = useContext(StatusContext);

  // 获取模型广场权限配置
  const pricingRequireAuth = useMemo(() => {
    const headerNavModulesConfig = statusState?.status?.HeaderNavModules;
    if (headerNavModulesConfig) {
      try {
        const modules = JSON.parse(headerNavModulesConfig);

        // 处理向后兼容性：如果pricing是boolean，默认不需要登录
        if (typeof modules.pricing === 'boolean') {
          return false; // 默认不需要登录鉴权
        }

        // 如果是对象格式，使用requireAuth配置
        return modules.pricing?.requireAuth === true;
      } catch (error) {
        console.error('解析顶栏模块配置失败:', error);
        return false; // 默认不需要登录
      }
    }
    return false; // 默认不需要登录
  }, [statusState?.status?.HeaderNavModules]);

  return (
    <SetupCheck>
      <Routes>
        <Route
          path='/'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <Home />
            </Suspense>
          }
        />
        <Route
          path='/setup'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <Setup />
            </Suspense>
          }
        />
        <Route path='/forbidden' element={<Forbidden />} />
        <Route
          path='/console/models'
          element={
            <PermissionRoute permissions={['ops.manage']}>
              <ModelPage />
            </PermissionRoute>
          }
        />
        <Route
          path='/console/deployment'
          element={
            <PermissionRoute permissions={['ops.manage']}>
              <ModelDeploymentPage />
            </PermissionRoute>
          }
        />
        <Route
          path='/console/subscription'
          element={
            <PermissionRoute permissions={['ops.manage']}>
              <Subscription />
            </PermissionRoute>
          }
        />
        <Route
          path='/console/billing'
          element={
            <PermissionRoute
              permissions={[
                'finance.view',
                'finance.write',
                'finance.audit.view',
                'system.manage',
              ]}
            >
              <Billing />
            </PermissionRoute>
          }
        />
        <Route
          path='/console/channel'
          element={
            <PermissionRoute permissions={['ops.manage']}>
              <Channel />
            </PermissionRoute>
          }
        />
        <Route
          path='/console/token'
          element={
            <PrivateRoute>
              <Token />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/playground'
          element={
            <PrivateRoute>
              <Playground />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/image-playground'
          element={
            <PrivateRoute>
              <ImagePlayground />
            </PrivateRoute>
          }
        />
        <Route
          path='/console/redemption'
          element={
            <PermissionRoute permissions={['finance.write', 'system.manage']}>
              <Redemption />
            </PermissionRoute>
          }
        />
        <Route
          path='/console/user'
          element={
            <PermissionRoute
              permissions={['ops.manage', 'finance.write', 'system.manage']}
            >
              <User />
            </PermissionRoute>
          }
        />
        <Route
          path='/user/reset'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PasswordResetConfirm />
            </Suspense>
          }
        />
        <Route
          path='/login'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <AuthRedirect>
                <LoginForm />
              </AuthRedirect>
            </Suspense>
          }
        />
        <Route
          path='/register'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <AuthRedirect>
                <RegisterForm />
              </AuthRedirect>
            </Suspense>
          }
        />
        <Route
          path='/reset'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PasswordResetForm />
            </Suspense>
          }
        />
        <Route
          path='/oauth/github'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <OAuth2Callback type='github'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/discord'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <OAuth2Callback type='discord'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/oidc'
          element={
            <Suspense fallback={<Loading></Loading>}>
              <OAuth2Callback type='oidc'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/linuxdo'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <OAuth2Callback type='linuxdo'></OAuth2Callback>
            </Suspense>
          }
        />
        <Route
          path='/oauth/:provider'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <DynamicOAuth2Callback />
            </Suspense>
          }
        />
        <Route
          path='/console/setting'
          element={
            <PermissionRoute permissions={['system.manage']}>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Setting />
              </Suspense>
            </PermissionRoute>
          }
        />
        <Route
          path='/console/personal'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <PersonalSetting />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/console/topup'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <TopUp />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/console/log'
          element={
            <PrivateRoute>
              <Log />
            </PrivateRoute>
          }
        />
        <Route
          path='/console'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Dashboard />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/console/midjourney'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <MjProxy />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/console/task'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Task />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route
          path='/pricing'
          element={
            pricingRequireAuth ? (
              <PrivateRoute>
                <Suspense
                  fallback={<Loading></Loading>}
                  key={location.pathname}
                >
                  <Pricing />
                </Suspense>
              </PrivateRoute>
            ) : (
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Pricing />
              </Suspense>
            )
          }
        />
        <Route
          path='/about'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <About />
            </Suspense>
          }
        />
        <Route
          path='/user-agreement'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <UserAgreement />
            </Suspense>
          }
        />
        <Route
          path='/privacy-policy'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <PrivacyPolicy />
            </Suspense>
          }
        />
        <Route
          path='/console/chat/:id?'
          element={
            <Suspense fallback={<Loading></Loading>} key={location.pathname}>
              <Chat />
            </Suspense>
          }
        />
        {/* 方便使用chat2link直接跳转聊天... */}
        <Route
          path='/chat2link'
          element={
            <PrivateRoute>
              <Suspense fallback={<Loading></Loading>} key={location.pathname}>
                <Chat2Link />
              </Suspense>
            </PrivateRoute>
          }
        />
        <Route path='*' element={<NotFound />} />
      </Routes>
    </SetupCheck>
  );
}

export default App;
