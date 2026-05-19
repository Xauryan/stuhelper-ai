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
import { Button } from '@douyinfe/semi-ui';
import { Link } from 'react-router-dom';
import { ArrowRight, BookOpen, Github } from 'lucide-react';
import openExternal from '../lib/openExternal';

const CTA = ({ t, isMobile, docsLink, isDemoSiteMode, version }) => {
  return (
    <section className='sh-section sh-reveal'>
      <div className='sh-cta sh-beam-card'>
        <h3 className='sh-cta-title'>
          {t('准备好把 AI 接入你的工作流了吗？')}
          <br />
          <span className='sh-cta-title-accent'>
            {t('一分钟，把 StuHelper AI 接进你的工具链。')}
          </span>
        </h3>
        <p className='sh-cta-desc'>
          {t(
            '无需绑卡，无需高额起步充值；注册即得专属 Key，立刻为你常用的工具装上 AI 引擎。',
          )}
        </p>
        <div className='sh-cta-actions'>
          <Link to='/console'>
            <Button
              theme='solid'
              type='primary'
              size={isMobile ? 'default' : 'large'}
              className='sh-btn-primary'
              icon={<ArrowRight />}
            >
              {t('立即注册使用')}
            </Button>
          </Link>
          {isDemoSiteMode && version ? (
            <Button
              size={isMobile ? 'default' : 'large'}
              className='sh-btn-ghost'
              icon={<Github size={16} />}
              onClick={() =>
                openExternal('https://github.com/Xauryan/stuhelper-ai')
              }
            >
              {version}
            </Button>
          ) : docsLink ? (
            <Button
              size={isMobile ? 'default' : 'large'}
              className='sh-btn-ghost'
              icon={<BookOpen size={16} />}
              onClick={() => openExternal(docsLink)}
            >
              {t('阅读完整文档')}
            </Button>
          ) : (
            <Link to='/console'>
              <Button
                size={isMobile ? 'default' : 'large'}
                className='sh-btn-ghost'
                icon={<BookOpen size={16} />}
              >
                {t('阅读完整文档')}
              </Button>
            </Link>
          )}
        </div>
      </div>
    </section>
  );
};

export default CTA;
