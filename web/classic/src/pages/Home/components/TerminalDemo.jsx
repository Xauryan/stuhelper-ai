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

import React, { useEffect, useMemo, useState } from 'react';
import { Button, Tooltip } from '@douyinfe/semi-ui';
import { IconCopy } from '@douyinfe/semi-icons';
import SectionHeader from './SectionHeader';
import useInViewOnce from '../hooks/useInViewOnce';
import { buildCurl } from '../lib/buildCurl';

const TYPING_SPEED = 14;

const usePrefersReducedMotion = () => {
  const [reduced, setReduced] = useState(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return false;
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  });

  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return undefined;
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    const onChange = (e) => setReduced(e.matches);
    if (mq.addEventListener) {
      mq.addEventListener('change', onChange);
      return () => mq.removeEventListener('change', onChange);
    }
    mq.addListener(onChange);
    return () => mq.removeListener(onChange);
  }, []);

  return reduced;
};

const TerminalDemo = ({ t, serverAddress, onCopyBaseURL, systemName }) => {
  const curl = useMemo(() => buildCurl(serverAddress), [serverAddress]);
  const comment = t('// 一行命令，把 {{name}} 接到你熟悉的工具', {
    name: systemName,
  });
  const fullScript = `${comment}\n${curl}`;
  const [ref, inView] = useInViewOnce({ threshold: 0.25 });
  const [typed, setTyped] = useState('');
  const reduced = usePrefersReducedMotion();

  useEffect(() => {
    if (!inView) return undefined;
    if (reduced) {
      setTyped(fullScript);
      return undefined;
    }
    let cancelled = false;
    let timeoutId = null;
    let i = 0;
    setTyped('');
    const tick = () => {
      if (cancelled) return;
      i += 1;
      setTyped(fullScript.slice(0, i));
      if (i < fullScript.length) {
        timeoutId = window.setTimeout(tick, TYPING_SPEED);
      }
    };
    timeoutId = window.setTimeout(tick, 120);
    return () => {
      cancelled = true;
      if (timeoutId !== null) window.clearTimeout(timeoutId);
    };
  }, [inView, fullScript, reduced]);

  const typingDone = typed.length >= fullScript.length;

  const renderLines = () => {
    const idx = typed.indexOf('\n');
    if (idx === -1) {
      return (
        <span className='sh-terminal-comment'>
          {typed}
          {!typingDone || !reduced ? (
            <span className='sh-cursor' aria-hidden='true' />
          ) : null}
        </span>
      );
    }
    const commentText = typed.slice(0, idx);
    const rest = typed.slice(idx + 1);
    return (
      <>
        <span className='sh-terminal-comment'>{commentText}</span>
        {'\n'}
        {rest}
        {!typingDone || !reduced ? (
          <span className='sh-cursor' aria-hidden='true' />
        ) : null}
      </>
    );
  };

  return (
    <section className='sh-section sh-reveal' ref={ref}>
      <div className='sh-terminal-wrap'>
        <div className='sh-terminal-copy'>
          <SectionHeader
            eyebrow={t('开发者接入')}
            title={t('把 SDK 指向 StuHelper 网关')}
          />
          <p>
            {t(
              '保留你现有的 OpenAI 兼容客户端，只替换 BASE_URL 与 API Key 即可。',
            )}
          </p>
          <p>
            {t(
              '终端、Notebook、IDE 插件、Web 客户端共用同一个密钥，体验保持一致。',
            )}
          </p>
        </div>
        <div className='sh-terminal' aria-label={t('请求示例')}>
          <div className='sh-terminal-bar'>
            <span className='sh-terminal-dots' aria-hidden='true'>
              <span />
              <span />
              <span />
            </span>
            <span className='sh-terminal-title'>~/stuhelper-ai $ curl</span>
            <Tooltip content={t('复制基础地址')}>
              <Button
                size='small'
                icon={<IconCopy />}
                onClick={onCopyBaseURL}
                className='sh-terminal-copy-btn'
                aria-label={t('复制基础地址')}
              />
            </Tooltip>
          </div>
          <pre>
            <code>{renderLines()}</code>
          </pre>
        </div>
      </div>
    </section>
  );
};

export default TerminalDemo;
