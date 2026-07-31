/*
Copyright (C) 2023-2026 QuantumNous

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
import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'

import { useStatus } from '@/hooks/use-status'

type EnterpriseHomeProps = {
  isAuthenticated: boolean
}

const providers = [
  { name: 'Azure', mark: <span className='ukx-mark ukx-mark-azure' /> },
  { name: 'AWS', mark: <span className='ukx-mark ukx-mark-aws'>AWS</span> },
  { name: 'GCP', mark: <span className='ukx-mark ukx-mark-gcp' /> },
  { name: 'DeepSeek', mark: <span className='ukx-mark ukx-mark-deepseek'>DS</span> },
  { name: '豆包', mark: <span className='ukx-mark ukx-mark-doubao'>豆</span> },
  { name: 'Kimi', mark: <span className='ukx-mark ukx-mark-kimi' /> },
  { name: 'xAI', mark: <span className='ukx-mark ukx-mark-xai'>xAI</span> },
]

function DocsLink(props: { className: string; children: ReactNode }) {
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'

  return (
    <a
      href={docsUrl}
      target={docsUrl.startsWith('http') ? '_blank' : undefined}
      rel='noreferrer'
      className={props.className}
    >
      {props.children}
    </a>
  )
}

export function EnterpriseHome({ isAuthenticated }: EnterpriseHomeProps) {
  return (
    <main className='ukx-home'>
      <style>{`
        .ukx-home {
          min-height: 100vh;
          background: #f7f9fc;
          color: #172033;
          font-family: Inter, "Segoe UI", "Microsoft YaHei", "PingFang SC", Arial, sans-serif;
        }

        .ukx-home-header > div {
          max-width: 1120px !important;
          padding-top: 10px !important;
        }

        .ukx-home-header nav {
          min-height: 52px !important;
          height: 52px !important;
          padding: 0 10px 0 14px !important;
          border: 1px solid rgba(255, 255, 255, 0.16);
          border-radius: 18px;
          background: rgba(8, 18, 32, 0.78) !important;
          box-shadow: 0 14px 36px rgba(3, 9, 18, 0.26), inset 0 1px 0 rgba(255, 255, 255, 0.08);
          color: rgba(255, 255, 255, 0.92);
          backdrop-filter: blur(18px) saturate(1.18);
        }

        .ukx-home-header nav a,
        .ukx-home-header nav span {
          color: rgba(255, 255, 255, 0.82);
        }

        .ukx-home-header nav a:hover,
        .ukx-home-header nav a[data-status="active"],
        .ukx-home-header nav .text-foreground {
          color: #fff !important;
        }

        .ukx-home-header nav .text-muted-foreground {
          color: rgba(255, 255, 255, 0.78) !important;
        }

        .ukx-home-header nav button {
          color: rgba(255, 255, 255, 0.9);
        }

        .ukx-home-header nav button:not([class*="bg-primary"]) {
          background: rgba(255, 255, 255, 0.08);
          border-color: rgba(255, 255, 255, 0.12);
        }

        .ukx-home-header nav [class*="bg-border"] {
          background: rgba(255, 255, 255, 0.18) !important;
        }

        .ukx-hero {
          position: relative;
          min-height: 76vh;
          display: grid;
          align-items: center;
          overflow: hidden;
          padding: 112px 48px 92px;
          color: #fff;
          background:
            linear-gradient(rgba(5, 18, 34, 0.72), rgba(5, 18, 34, 0.72)),
            url("https://images.unsplash.com/photo-1558494949-ef010cbdcc31?auto=format&fit=crop&w=2400&q=80") center / cover;
        }

        .ukx-hero::before {
          content: "";
          position: absolute;
          inset: 0;
          z-index: 0;
          pointer-events: none;
          background: linear-gradient(90deg, rgba(70, 200, 255, 0.08), transparent 28%, transparent 72%, rgba(78, 230, 186, 0.07));
          mix-blend-mode: screen;
          opacity: 0.75;
        }

        .ukx-hero::after {
          content: "";
          position: absolute;
          inset: -10% 0;
          z-index: 0;
          pointer-events: none;
          background:
            radial-gradient(circle at 8% 22%, rgba(122, 236, 255, 0.95) 0 1px, rgba(122, 236, 255, 0.36) 2px, transparent 5px),
            radial-gradient(circle at 20% 66%, rgba(77, 255, 200, 0.9) 0 1px, rgba(77, 255, 200, 0.32) 2px, transparent 5px),
            radial-gradient(circle at 36% 40%, rgba(255, 255, 255, 0.92) 0 1px, rgba(122, 236, 255, 0.28) 2px, transparent 5px),
            radial-gradient(circle at 52% 76%, rgba(122, 236, 255, 0.8) 0 1px, rgba(122, 236, 255, 0.26) 2px, transparent 5px),
            radial-gradient(circle at 68% 30%, rgba(77, 255, 200, 0.78) 0 1px, rgba(77, 255, 200, 0.25) 2px, transparent 5px),
            radial-gradient(circle at 84% 58%, rgba(255, 255, 255, 0.86) 0 1px, rgba(122, 236, 255, 0.25) 2px, transparent 5px),
            radial-gradient(circle at 96% 24%, rgba(77, 255, 200, 0.72) 0 1px, rgba(77, 255, 200, 0.22) 2px, transparent 5px),
            repeating-linear-gradient(90deg, transparent 0 86px, rgba(120, 228, 255, 0.07) 86px 88px, transparent 88px 180px);
          background-size: 1000px 100%, 920px 100%, 1080px 100%, 860px 100%, 1040px 100%, 940px 100%, 1180px 100%, 520px 100%;
          mix-blend-mode: screen;
          opacity: 0.88;
          animation: ukx-light-speed 3.8s linear infinite;
        }

        .ukx-dart-layer {
          position: absolute;
          inset: 0;
          z-index: 0;
          overflow: hidden;
          pointer-events: none;
        }

        .ukx-dart {
          position: absolute;
          width: 20px;
          height: 20px;
          clip-path: polygon(50% 0, 62% 38%, 100% 50%, 62% 62%, 50% 100%, 38% 62%, 0 50%, 38% 38%);
          background: radial-gradient(circle, #ffffff 0 10%, #54f3c2 22%, #47b8ff 62%, transparent 68%);
          box-shadow: 0 0 16px rgba(84, 243, 194, 0.55), 0 0 34px rgba(71, 184, 255, 0.28);
          opacity: 0;
        }

        .ukx-dart::before {
          content: "";
          position: absolute;
          inset: 5px;
          border-radius: 99px;
          background: rgba(255, 255, 255, 0.82);
          filter: blur(1px);
        }

        .ukx-dart.one { top: 18%; left: -8%; animation: ukx-dart-left-right 4.6s linear infinite; }
        .ukx-dart.two {
          top: 72%;
          right: -8%;
          background: radial-gradient(circle, #ffffff 0 10%, #ffcc66 22%, #ff5ca8 62%, transparent 68%);
          box-shadow: 0 0 16px rgba(255, 92, 168, 0.5), 0 0 34px rgba(255, 204, 102, 0.24);
          animation: ukx-dart-right-left 5.2s linear infinite 0.6s;
        }
        .ukx-dart.three {
          top: -10%;
          left: 72%;
          background: radial-gradient(circle, #ffffff 0 10%, #8d7cff 22%, #4debd0 62%, transparent 68%);
          box-shadow: 0 0 16px rgba(141, 124, 255, 0.48), 0 0 34px rgba(77, 235, 208, 0.24);
          animation: ukx-dart-top-bottom 5.8s linear infinite 1.1s;
        }
        .ukx-dart.four {
          bottom: -10%;
          left: 24%;
          background: radial-gradient(circle, #ffffff 0 10%, #74d8ff 22%, #5cff9f 62%, transparent 68%);
          animation: ukx-dart-bottom-top 5s linear infinite 1.8s;
        }

        .ukx-hero-inner {
          position: relative;
          z-index: 1;
          width: min(1080px, 100%);
          margin: 0 auto;
        }

        .ukx-eyebrow {
          display: inline-flex;
          align-items: center;
          gap: 8px;
          margin-bottom: 24px;
          color: rgba(255, 255, 255, 0.78);
          font-size: 14px;
          font-weight: 600;
        }

        .ukx-eyebrow::before {
          content: "";
          width: 8px;
          height: 8px;
          border-radius: 99px;
          background: #3bd3a4;
        }

        .ukx-title {
          max-width: 760px;
          margin: 0;
          font-size: 56px;
          line-height: 1.1;
          font-weight: 760;
          letter-spacing: 0;
        }

        .ukx-lead {
          max-width: 720px;
          margin: 24px 0 0;
          color: rgba(255, 255, 255, 0.84);
          font-size: 20px;
          line-height: 1.7;
        }

        .ukx-mission {
          max-width: 720px;
          margin: 18px 0 0;
          color: rgba(255, 255, 255, 0.92);
          font-size: 18px;
          line-height: 1.6;
          font-weight: 680;
        }

        .ukx-actions {
          display: flex;
          flex-wrap: wrap;
          gap: 12px;
          margin-top: 34px;
        }

        .ukx-button {
          display: inline-flex;
          align-items: center;
          justify-content: center;
          min-height: 44px;
          padding: 0 20px;
          border-radius: 7px;
          font-size: 15px;
          font-weight: 650;
          text-decoration: none;
        }

        .ukx-button.primary { background: #fff; color: #10243a; }
        .ukx-button.secondary { border: 1px solid rgba(255, 255, 255, 0.42); color: #fff; }

        .ukx-entry,
        .ukx-trust {
          width: min(1080px, calc(100% - 48px));
          margin-left: auto;
          margin-right: auto;
          border: 1px solid #dbe2ea;
          border-radius: 8px;
          background: #fff;
        }

        .ukx-entry {
          position: relative;
          z-index: 2;
          display: grid;
          grid-template-columns: 1fr auto;
          align-items: center;
          gap: 24px;
          margin-top: -42px;
          padding: 28px;
          box-shadow: 0 18px 50px rgba(24, 39, 75, 0.08);
        }

        .ukx-entry h2 {
          margin: 0 0 8px;
          font-size: 26px;
          line-height: 1.25;
          font-weight: 760;
        }

        .ukx-entry p {
          margin: 0;
          color: #5d687a;
          line-height: 1.7;
        }

        .ukx-entry-actions {
          display: flex;
          flex-wrap: wrap;
          justify-content: flex-end;
          gap: 10px;
        }

        .ukx-entry-actions .primary { background: #1267b3; color: #fff; }
        .ukx-entry-actions .secondary { border: 1px solid #dbe2ea; color: #0d4f88; background: #fff; }

        .ukx-trust {
          display: grid;
          gap: 24px;
          margin-top: 22px;
          padding: 22px 28px;
        }

        .ukx-trust-title {
          margin: 0 0 6px;
          font-size: 16px;
          font-weight: 720;
        }

        .ukx-trust-copy {
          margin: 0;
          color: #5d687a;
          font-size: 14px;
          line-height: 1.7;
        }

        .ukx-logo-wall {
          display: grid;
          grid-template-columns: repeat(7, minmax(96px, 1fr));
          gap: 12px;
        }

        .ukx-logo-card {
          min-height: 64px;
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 9px;
          padding: 10px 12px;
          border: 1px solid #dbe2ea;
          border-radius: 7px;
          background: #f8fbff;
          color: #17324d;
        }

        .ukx-logo-card span:last-child {
          font-size: 14px;
          font-weight: 720;
          white-space: nowrap;
        }

        .ukx-mark {
          position: relative;
          display: inline-grid;
          width: 26px;
          height: 26px;
          flex: 0 0 26px;
          place-items: center;
          font-size: 11px;
          font-weight: 800;
        }

        .ukx-mark-azure { width: 24px; height: 22px; }
        .ukx-mark-azure::before {
          content: "";
          position: absolute;
          inset: 0;
          background: #0078d4;
          clip-path: polygon(45% 0, 100% 100%, 59% 100%, 46% 71%, 29% 100%, 0 100%);
        }

        .ukx-mark-aws { color: #232f3e; font-size: 13px; }
        .ukx-mark-aws::after {
          content: "";
          position: absolute;
          left: 4px;
          right: 2px;
          bottom: 1px;
          height: 7px;
          border-bottom: 3px solid #ff9900;
          border-radius: 0 0 30px 30px;
          transform: rotate(-7deg);
        }

        .ukx-mark-gcp { width: 28px; height: 20px; }
        .ukx-mark-gcp::before {
          content: "";
          position: absolute;
          inset: 2px 1px;
          border: 5px solid #4285f4;
          border-top-color: #ea4335;
          border-right-color: #fbbc04;
          border-left-color: #34a853;
          border-radius: 14px;
        }

        .ukx-mark-deepseek { border-radius: 8px; color: #fff; background: #2f7ff7; }
        .ukx-mark-doubao { border-radius: 8px; color: #fff; background: linear-gradient(135deg, #2f6df6, #7a5cff 55%, #fb5aa6); }
        .ukx-mark-kimi { border-radius: 99px; color: #fff; background: #111827; }
        .ukx-mark-kimi::before {
          content: "";
          position: absolute;
          width: 12px;
          height: 12px;
          border-radius: 99px;
          border: 2px solid #fff;
          border-left-color: transparent;
          transform: rotate(-25deg);
        }
        .ukx-mark-xai { color: #111827; font-size: 14px; }

        .ukx-principles {
          width: min(1080px, calc(100% - 48px));
          margin: 0 auto;
          padding: 46px 0 42px;
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 18px;
        }

        .ukx-principle {
          min-height: 168px;
          padding: 24px;
          border: 1px solid #dbe2ea;
          border-radius: 8px;
          background: #fff;
        }

        .ukx-principle-index { color: #18866f; font-size: 13px; font-weight: 800; }
        .ukx-principle h3 { margin: 16px 0 10px; font-size: 19px; line-height: 1.3; font-weight: 760; }
        .ukx-principle p { margin: 0; color: #5d687a; font-size: 15px; line-height: 1.75; }

        .ukx-footer {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 20px;
          min-height: 72px;
          padding: 0 48px;
          color: rgba(255, 255, 255, 0.78);
          background: #15171d;
          font-size: 14px;
        }

        .ukx-footer a { color: #66a9ff; font-weight: 650; text-decoration: none; }

        @keyframes ukx-light-speed {
          0% { background-position: 0 0, 0 0, 0 0, 0 0, 0 0, 0 0, 0 0, 0 0; }
          100% { background-position: 1000px 0, -920px 0, 1080px 0, -860px 0, 1040px 0, -940px 0, 1180px 0, 520px 0; }
        }

        @keyframes ukx-dart-left-right {
          0% { opacity: 0; transform: translate3d(0, 0, 0) rotate(35deg) scale(0.82); }
          10%, 82% { opacity: 0.9; }
          100% { opacity: 0; transform: translate3d(118vw, 22vh, 0) rotate(35deg) scale(1); }
        }
        @keyframes ukx-dart-right-left {
          0% { opacity: 0; transform: translate3d(0, 0, 0) rotate(205deg) scale(0.82); }
          10%, 80% { opacity: 0.82; }
          100% { opacity: 0; transform: translate3d(-118vw, -26vh, 0) rotate(205deg) scale(1); }
        }
        @keyframes ukx-dart-top-bottom {
          0% { opacity: 0; transform: translate3d(0, 0, 0) rotate(112deg) scale(0.78); }
          12%, 78% { opacity: 0.78; }
          100% { opacity: 0; transform: translate3d(-44vw, 112vh, 0) rotate(112deg) scale(1); }
        }
        @keyframes ukx-dart-bottom-top {
          0% { opacity: 0; transform: translate3d(0, 0, 0) rotate(-68deg) scale(0.78); }
          12%, 80% { opacity: 0.78; }
          100% { opacity: 0; transform: translate3d(52vw, -112vh, 0) rotate(-68deg) scale(1); }
        }

        @media (max-width: 820px) {
          .ukx-hero { min-height: 72vh; padding: 92px 24px 82px; }
          .ukx-title { font-size: 38px; }
          .ukx-lead { font-size: 17px; }
          .ukx-mission { font-size: 16px; }
          .ukx-entry, .ukx-trust, .ukx-principles { width: calc(100% - 32px); }
          .ukx-entry { grid-template-columns: 1fr; margin-top: -32px; padding: 22px; }
          .ukx-entry-actions { justify-content: flex-start; }
          .ukx-logo-wall { grid-template-columns: repeat(2, minmax(0, 1fr)); }
          .ukx-logo-card { justify-content: flex-start; }
          .ukx-principles { grid-template-columns: 1fr; }
          .ukx-footer {
            min-height: 96px;
            flex-direction: column;
            align-items: flex-start;
            justify-content: center;
            padding: 20px 24px;
          }
        }

        @media (prefers-reduced-motion: reduce) {
          .ukx-hero::after,
          .ukx-dart { animation: none; }
        }
      `}</style>

      <section className='ukx-hero'>
        <div className='ukx-dart-layer' aria-hidden='true'>
          <span className='ukx-dart one' />
          <span className='ukx-dart two' />
          <span className='ukx-dart three' />
          <span className='ukx-dart four' />
        </div>
        <div className='ukx-hero-inner'>
          <div className='ukx-eyebrow'>Enterprise AI Model Service</div>
          <h1 className='ukx-title'>稳定的企业级原厂模型服务</h1>
          <p className='ukx-lead'>
            UniKeyX 依托 Azure、AWS 与 GCP 云基础设施，提供纯净、可靠、长期可用的大模型接入服务，满足企业 7x24 小时持续使用需求。
          </p>
          <p className='ukx-mission'>
            我们的宗旨：模型跑得稳，服务靠得住，用户用得起。
          </p>
          <div className='ukx-actions'>
            {isAuthenticated ? (
              <Link className='ukx-button primary' to='/dashboard'>
                进入控制台
              </Link>
            ) : (
              <>
                <Link className='ukx-button primary' to='/register'>
                  注册账号
                </Link>
                <Link className='ukx-button secondary' to='/sign-in'>
                  登录
                </Link>
              </>
            )}
            <DocsLink className='ukx-button secondary'>查看文档</DocsLink>
          </div>
        </div>
      </section>

      <section className='ukx-entry' aria-label='进入系统'>
        <div>
          <h2>把 AI 能力稳定接入你的业务</h2>
          <p>适合需要长期稳定调用大模型的企业、团队和专业用户。</p>
        </div>
        <div className='ukx-entry-actions'>
          {isAuthenticated ? (
            <Link className='ukx-button primary' to='/dashboard'>
              进入控制台
            </Link>
          ) : (
            <>
              <Link className='ukx-button primary' to='/sign-in'>
                登录
              </Link>
              <Link className='ukx-button secondary' to='/register'>
                注册账号
              </Link>
            </>
          )}
          <Link className='ukx-button secondary' to='/pricing'>
            查看模型价格
          </Link>
        </div>
      </section>

      <section className='ukx-trust' aria-label='国内外模型与云基础设施'>
        <div>
          <p className='ukx-trust-title'>整合国内外原厂模型与主流云基础设施</p>
          <p className='ukx-trust-copy'>
            背靠 Azure、AWS、GCP 等云平台，连接 DeepSeek、豆包、Kimi、xAI 等国内外模型能力，为企业提供稳定、清晰、可持续的统一接入服务。
          </p>
        </div>
        <div className='ukx-logo-wall' aria-label='云平台与模型品牌'>
          {providers.map((provider) => (
            <span className='ukx-logo-card' key={provider.name}>
              {provider.mark}
              <span>{provider.name}</span>
            </span>
          ))}
        </div>
      </section>

      <section className='ukx-principles'>
        <article className='ukx-principle'>
          <div className='ukx-principle-index'>01</div>
          <h3>真模型</h3>
          <p>连接国内外原厂模型能力，少一点包装，多一点透明，让每次调用都清清楚楚。</p>
        </article>
        <article className='ukx-principle'>
          <div className='ukx-principle-index'>02</div>
          <h3>稳服务</h3>
          <p>背靠多云基础设施和长期服务协议，面向企业 7x24 小时持续运行。</p>
        </article>
        <article className='ukx-principle'>
          <div className='ukx-principle-index'>03</div>
          <h3>长久用</h3>
          <p>不做短期噱头，不透支信任。踏踏实实把服务做好，让客户放心把业务交给我们。</p>
        </article>
      </section>

      <footer className='ukx-footer'>
        <span>© 2026 UniKeyX API. 版权所有</span>
        <span>
          设计与开发由{' '}
          <a
            href='https://github.com/QuantumNous/new-api'
            target='_blank'
            rel='noopener noreferrer'
          >
            New API
          </a>
        </span>
      </footer>
    </main>
  )
}
