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
import {
  Aws,
  Azure,
  DeepSeek,
  Doubao,
  GoogleCloud,
  Kimi,
  XAI,
} from '@lobehub/icons'
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  BookOpen,
  Boxes,
  Infinity as InfinityIcon,
  LogIn,
  ShieldCheck,
  UserPlus,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import homeImage800 from '@/assets/branding/unikeyx-home-800.webp'
import homeImage1600 from '@/assets/branding/unikeyx-home-1600.webp'
import homeImage2400 from '@/assets/branding/unikeyx-home-2400.webp'
import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

import '../../enterprise-home.css'

const providers = [
  { name: 'Azure', icon: Azure.Color },
  { name: 'AWS', icon: Aws.Color },
  { name: 'GCP', icon: GoogleCloud.Color },
  { name: 'DeepSeek', icon: DeepSeek.Color },
  { name: 'Doubao', icon: Doubao.Color },
  { name: 'Kimi', icon: Kimi },
  { name: 'xAI', icon: XAI },
]

export function EnterpriseHome(props: { isAuthenticated: boolean }) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    typeof status?.docs_link === 'string' && status.docs_link
      ? status.docs_link
      : 'https://doc.unikeyx.com'

  return (
    <main className='unikeyx-home'>
      <section className='unikeyx-hero'>
        <img
          className='unikeyx-hero-image'
          src={homeImage1600}
          srcSet={`${homeImage800} 800w, ${homeImage1600} 1600w, ${homeImage2400} 2400w`}
          sizes='100vw'
          width={1600}
          height={898}
          alt=''
          fetchPriority='high'
        />
        <div className='unikeyx-hero-inner'>
          <p className='unikeyx-eyebrow'>{t('Enterprise AI Model Service')}</p>
          <h1>UniKeyX API</h1>
          <h2>{t('Reliable enterprise-grade original model services')}</h2>
          <p className='unikeyx-lead'>
            {t(
              'UniKeyX uses Azure, AWS and GCP cloud infrastructure to provide clean, reliable, long-term model access for enterprise workloads running 24/7.'
            )}
          </p>
          <p className='unikeyx-mission'>
            {t(
              'Our mission: reliable models, dependable service, affordable access.'
            )}
          </p>
          <div className='unikeyx-actions'>
            {props.isAuthenticated ? (
              <Button
                size='lg'
                className='unikeyx-action-primary'
                render={<Link to='/dashboard' />}
              >
                {t('Enter console')}
                <ArrowRight data-icon='inline-end' />
              </Button>
            ) : (
              <>
                <Button
                  size='lg'
                  className='unikeyx-action-primary'
                  render={<Link to='/sign-up' />}
                >
                  <UserPlus data-icon='inline-start' />
                  {t('Register account')}
                </Button>
                <Button
                  size='lg'
                  variant='outline'
                  render={<Link to='/sign-in' />}
                >
                  <LogIn data-icon='inline-start' />
                  {t('Sign in')}
                </Button>
              </>
            )}
            <Button
              size='lg'
              variant='outline'
              render={
                <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
              }
            >
              <BookOpen data-icon='inline-start' />
              {t('View documentation')}
            </Button>
          </div>
        </div>
      </section>

      <section className='unikeyx-entry'>
        <div className='unikeyx-container unikeyx-entry-inner'>
          <div>
            <h2>{t('Connect AI reliably to your business')}</h2>
            <p>
              {t(
                'For enterprises, teams and professionals who need dependable long-term model access.'
              )}
            </p>
          </div>
          <div className='unikeyx-actions'>
            {props.isAuthenticated ? (
              <Button size='lg' render={<Link to='/dashboard' />}>
                {t('Enter console')}
                <ArrowRight data-icon='inline-end' />
              </Button>
            ) : (
              <>
                <Button size='lg' render={<Link to='/sign-in' />}>
                  <LogIn data-icon='inline-start' />
                  {t('Sign in')}
                </Button>
                <Button
                  size='lg'
                  variant='outline'
                  render={<Link to='/sign-up' />}
                >
                  <UserPlus data-icon='inline-start' />
                  {t('Register account')}
                </Button>
              </>
            )}
            <Button size='lg' variant='outline' render={<Link to='/pricing' />}>
              {t('View model pricing')}
              <ArrowRight data-icon='inline-end' />
            </Button>
          </div>
        </div>
      </section>

      <section className='unikeyx-trust unikeyx-container'>
        <div className='unikeyx-trust-heading'>
          <h2>
            {t('Original models and leading cloud infrastructure worldwide')}
          </h2>
          <p>
            {t(
              'Built on Azure, AWS and GCP, connecting DeepSeek, Doubao, Kimi and xAI through a stable, transparent and sustainable unified service.'
            )}
          </p>
        </div>
        <ul
          className='unikeyx-providers'
          aria-label={t('Cloud platforms and model providers')}
        >
          {providers.map((provider) => {
            const Icon = provider.icon
            return (
              <li key={provider.name}>
                <Icon size={32} aria-hidden='true' />
                <span>
                  {provider.name === 'Doubao' ? t('Doubao') : provider.name}
                </span>
              </li>
            )
          })}
        </ul>
      </section>

      <section className='unikeyx-principles unikeyx-container'>
        <article>
          <div className='unikeyx-principle-heading'>
            <span className='unikeyx-principle-index'>01</span>
            <Boxes size={24} aria-hidden='true' />
          </div>
          <h3>{t('Original models')}</h3>
          <p>
            {t(
              'Connect directly to original models worldwide. Less packaging, more transparency, and clarity in every request.'
            )}
          </p>
        </article>
        <article>
          <div className='unikeyx-principle-heading'>
            <span className='unikeyx-principle-index'>02</span>
            <ShieldCheck size={24} aria-hidden='true' />
          </div>
          <h3>{t('Reliable service')}</h3>
          <p>
            {t(
              'Multi-cloud infrastructure and long-term service agreements support enterprise operations around the clock.'
            )}
          </p>
        </article>
        <article>
          <div className='unikeyx-principle-heading'>
            <span className='unikeyx-principle-index'>03</span>
            <InfinityIcon size={24} aria-hidden='true' />
          </div>
          <h3>{t('Long-term access')}</h3>
          <p>
            {t(
              'No short-term gimmicks and no compromises on trust. Consistent service that customers can depend on for their business.'
            )}
          </p>
        </article>
      </section>
    </main>
  )
}
