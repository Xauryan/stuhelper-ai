/*
Copyright (C) 2025 Xauryan

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

For commercial licensing, please contact support@xauryan.com
*/

import React from 'react';
import { Link } from 'react-router-dom';
import SectionHeader from './SectionHeader';

const HowItWorks = ({ t, docsLink }) => {
  const steps = [
    {
      step: '01',
      label: t('开始'),
      title: t('注册账号'),
      desc: t('邮箱一键注册，登录即可领取体验额度，无需绑定信用卡。'),
      cta: t('前往控制台'),
      to: '/console',
      external: false,
    },
    {
      step: '02',
      label: t('配置'),
      title: t('接入客户端'),
      desc: t('将 API Key 与基础地址填入 Cursor、Cherry Studio 等熟悉的工具。'),
      cta: t('打开令牌管理'),
      to: '/console/token',
      external: false,
    },
    {
      step: '03',
      label: t('使用'),
      title: t('开始调用'),
      desc: t('让 AI 辅助开发、研究与创作，把 token 用在刀刃上。'),
      cta: t('打开使用日志'),
      to: '/console/log',
      external: false,
    },
  ];

  return (
    <section className='sh-section sh-reveal'>
      <SectionHeader
        eyebrow={t('快速上手向导')}
        title={t('三步开启你的 AI 工作流')}
        description={t(
          '我们梳理了常见接入路径，剔除冗余步骤，几分钟内即可完成配置并开始调用。',
        )}
      />
      <div className='sh-quickpath'>
        {steps.map(({ step, label, title, desc, cta, to, external }) => {
          const Inner = (
            <>
              <span className='sh-quickpath-step'>
                <span className='sh-quickpath-step-num'>{step}</span>
                {label}
              </span>
              <h3 className='sh-quickpath-title'>{title}</h3>
              <p className='sh-quickpath-desc'>{desc}</p>
              <span className='sh-quickpath-cta'>
                {cta}
                <span className='sh-quickpath-cta-arrow' aria-hidden='true'>
                  →
                </span>
              </span>
            </>
          );

          if (external) {
            return (
              <a
                key={step}
                href={to}
                target='_blank'
                rel='noopener noreferrer'
                className='sh-quickpath-card sh-beam-card'
              >
                {Inner}
              </a>
            );
          }

          return (
            <Link key={step} to={to} className='sh-quickpath-card sh-beam-card'>
              {Inner}
            </Link>
          );
        })}
      </div>
    </section>
  );
};

export default HowItWorks;
