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

import React, { useEffect, useState } from 'react';
import { Button, Tooltip } from '@douyinfe/semi-ui';
import { IconCopy } from '@douyinfe/semi-icons';
import { Link } from 'react-router-dom';
import {
  Sparkles,
  Zap,
  Plug,
  Receipt,
  ArrowRight,
  BookOpen,
} from 'lucide-react';
import openExternal from '../lib/openExternal';

const EndpointCycler = ({ items, index }) => {
  const [activeIdx, setActiveIdx] = useState(index);
  const [leavingIdx, setLeavingIdx] = useState(null);

  useEffect(() => {
    if (index === activeIdx) return undefined;
    setLeavingIdx(activeIdx);
    setActiveIdx(index);
    const timer = setTimeout(() => setLeavingIdx(null), 500);
    return () => clearTimeout(timer);
  }, [index, activeIdx]);

  return (
    <span className='sh-endpoint-cycler' aria-live='polite' aria-atomic='true'>
      {items.map((item, i) => {
        const cls =
          i === activeIdx ? 'is-active' : i === leavingIdx ? 'is-leaving' : '';
        return (
          <span
            key={item.value}
            className={`sh-endpoint-item ${cls}`}
            aria-hidden={i !== activeIdx}
          >
            {item.value}
          </span>
        );
      })}
    </span>
  );
};

const Hero = ({
  t,
  isMobile,
  serverAddress,
  endpointItems,
  endpointIndex,
  onCopyBaseURL,
  isDemoSiteMode,
  docsLink,
  version,
}) => {
  return (
    <section className='sh-hero sh-reveal'>
      <span className='sh-hero-kicker'>
        <Sparkles aria-hidden='true' />
        <span>{t('STUHELPER · AI GATEWAY')}</span>
      </span>

      <h1 className='sh-brand-title'>
        <span className='sh-brand-text'>StuHelper AI</span>
      </h1>
      <p className='sh-brand-tagline'>{t('统一的大模型 API 网关')}</p>

      <p className='sh-hero-subtitle'>
        {t(
          '一个 Key 接入主流大模型，更好的价格、更稳的接入。只需将客户端基址替换为：',
        )}
      </p>

      <ul className='sh-hero-badges' role='list'>
        <li className='sh-hero-badge'>
          <span className='sh-hero-badge-icon'>
            <Zap size={14} aria-hidden='true' />
          </span>
          {t('毫秒级响应')}
        </li>
        <li className='sh-hero-badge'>
          <span className='sh-hero-badge-icon'>
            <Plug size={14} aria-hidden='true' />
          </span>
          {t('OpenAI 协议兼容')}
        </li>
        <li className='sh-hero-badge'>
          <span className='sh-hero-badge-icon'>
            <Receipt size={14} aria-hidden='true' />
          </span>
          {t('透明计费')}
        </li>
      </ul>

      <div className='sh-base-panel'>
        <span className='sh-base-prefix'>BASE_URL</span>
        <span className='sh-base-url' title={serverAddress}>
          {serverAddress}
        </span>
        <EndpointCycler items={endpointItems} index={endpointIndex} />
        <Tooltip content={t('复制基础地址')}>
          <Button
            type='primary'
            onClick={onCopyBaseURL}
            icon={<IconCopy />}
            className='sh-btn-primary sh-base-copy'
            aria-label={t('复制基础地址')}
          />
        </Tooltip>
      </div>

      <div className='sh-hero-actions'>
        <Link to='/console'>
          <Button
            theme='solid'
            type='primary'
            size={isMobile ? 'default' : 'large'}
            className='sh-btn-primary'
            icon={<ArrowRight />}
          >
            {t('获取 API Key')}
          </Button>
        </Link>
        {isDemoSiteMode && version ? (
          <Button
            size={isMobile ? 'default' : 'large'}
            className='sh-btn-ghost'
            icon={<BookOpen size={16} />}
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
            {t('接入文档')}
          </Button>
        ) : (
          <Link to='/console'>
            <Button
              size={isMobile ? 'default' : 'large'}
              className='sh-btn-ghost'
              icon={<BookOpen size={16} />}
            >
              {t('接入文档')}
            </Button>
          </Link>
        )}
      </div>
    </section>
  );
};

export default Hero;
