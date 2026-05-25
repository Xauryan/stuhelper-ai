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

import React, { useCallback, useRef } from 'react';
import SectionHeader from './SectionHeader';

const FeatureGrid = ({ t, systemName }) => {
  const features = [
    {
      num: '01',
      title: t('无需代理，国内直连'),
      desc: t(
        '普通公网即可稳定调用，无需折腾代理。专为国内调用场景做了路由优化。',
      ),
      tag: t('低延迟'),
    },
    {
      num: '02',
      title: t('更优的算力定价'),
      desc: t('集中采买分摊成本，远低于官方标价；按用量计费，不必整年订阅。'),
      tag: t('按量计费'),
    },
    {
      num: '03',
      title: t('适配多种使用场景'),
      desc: t(
        '编程协作、科研阅读、写作翻译、办公总结——按需挑选模型，让一个 Key 覆盖你的日常。',
      ),
      tag: t('多场景'),
    },
    {
      num: '04',
      title: t('一个 Key 接入所有工具'),
      desc: t(
        '支持 Claude Code、Codex、Cursor、OpenCode、VSCode 扩展等主流客户端，一个 Key 全部打通。',
      ),
      tag: t('全平台'),
    },
  ];

  const rafIdRef = useRef(0);

  const onMouseMove = useCallback((e) => {
    const target = e.currentTarget;
    const rect = target.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    if (rafIdRef.current) cancelAnimationFrame(rafIdRef.current);
    rafIdRef.current = requestAnimationFrame(() => {
      target.style.setProperty('--sh-mx', `${x}px`);
      target.style.setProperty('--sh-my', `${y}px`);
    });
  }, []);

  return (
    <section className='sh-section sh-reveal'>
      <SectionHeader
        eyebrow={t('为什么选 {{name}}', { name: systemName })}
        title={t('更顺手的 API 中转站，专注稳定与性价比')}
        description={t(
          '我们把模型聚合、网络打通、计费透明这些重活揽下来，让你只关心提示词。',
        )}
      />
      <div className='sh-features'>
        {features.map(({ num, title, desc, tag }) => (
          <article
            key={num}
            className='sh-feature-row'
            onMouseMove={onMouseMove}
          >
            <div className='sh-feature-num' aria-hidden='true'>
              {num}
            </div>
            <div className='sh-feature-body'>
              <h3 className='sh-feature-title'>{title}</h3>
              <p className='sh-feature-desc'>{desc}</p>
            </div>
            <span className='sh-feature-tag'>{tag}</span>
          </article>
        ))}
      </div>
    </section>
  );
};

export default FeatureGrid;
