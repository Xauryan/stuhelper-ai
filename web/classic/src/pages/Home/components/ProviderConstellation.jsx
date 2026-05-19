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
import { Tooltip } from '@douyinfe/semi-ui';
import {
  Moonshot,
  OpenAI,
  XAI,
  Zhipu,
  Volcengine,
  Cohere,
  Claude,
  Gemini,
  Suno,
  Minimax,
  Wenxin,
  Spark,
  Qingyan,
  DeepSeek,
  Qwen,
  Midjourney,
  Grok,
  AzureAI,
  Hunyuan,
} from '@lobehub/icons';
import {
  Terminal,
  MessageSquare,
  Code2,
  Boxes,
  Globe,
  Layers,
} from 'lucide-react';
import SectionHeader from './SectionHeader';

const MODELS = [
  { label: 'OpenAI', Icon: OpenAI },
  { label: 'Claude', Icon: Claude.Color },
  { label: 'Gemini', Icon: Gemini.Color },
  { label: 'DeepSeek', Icon: DeepSeek.Color },
  { label: 'Qwen', Icon: Qwen.Color },
  { label: 'Zhipu', Icon: Zhipu.Color },
  { label: 'Moonshot', Icon: Moonshot },
  { label: 'Minimax', Icon: Minimax.Color },
  { label: 'Volcengine', Icon: Volcengine.Color },
  { label: 'Spark', Icon: Spark.Color },
  { label: 'Wenxin', Icon: Wenxin.Color },
  { label: 'Hunyuan', Icon: Hunyuan.Color },
  { label: 'Grok', Icon: Grok },
  { label: 'xAI', Icon: XAI },
  { label: 'Cohere', Icon: Cohere.Color },
  { label: 'Azure AI', Icon: AzureAI.Color },
  { label: 'Qingyan', Icon: Qingyan.Color },
  { label: 'Midjourney', Icon: Midjourney },
  { label: 'Suno', Icon: Suno },
];

const CLIENT_ICONS = {
  'Claude Code': Terminal,
  Codex: Terminal,
  Cursor: Code2,
  OpenCode: Code2,
  'VSCode 扩展': Layers,
  Cline: Terminal,
  'Cherry Studio': MessageSquare,
  Dify: Layers,
  FastGPT: Boxes,
  OpenWebUI: Globe,
  NextChat: MessageSquare,
  Ollama: Boxes,
  Chatbox: MessageSquare,
};

const CLIENTS = Object.keys(CLIENT_ICONS);

const ProviderConstellation = ({ t }) => {
  return (
    <section className='sh-section sh-reveal'>
      <SectionHeader
        eyebrow={t('生态覆盖范围')}
        title={t('主流模型与客户端，一个 Key 全打通')}
        description={t(
          '集结全球与国内顶尖大模型供应商，覆盖你日常使用的 IDE 插件、终端与桌面客户端。',
        )}
      />
      <div className='sh-ecosystem'>
        <div className='sh-eco-panel sh-beam-card'>
          <div className='sh-eco-head'>
            <h3 className='sh-eco-title'>{t('主流模型')}</h3>
            <span className='sh-eco-count'>{t('30+ 供应商')}</span>
          </div>
          <ul className='sh-eco-grid' role='list'>
            {MODELS.map(({ label, Icon }) => (
              <li key={label}>
                <Tooltip content={label}>
                  <span
                    className='sh-eco-item'
                    aria-label={label}
                    tabIndex={0}
                    role='button'
                  >
                    <span className='sh-eco-icon'>
                      <Icon size={20} />
                    </span>
                    <span className='sh-eco-label'>{label}</span>
                  </span>
                </Tooltip>
              </li>
            ))}
          </ul>
        </div>

        <div className='sh-eco-panel sh-beam-card'>
          <div className='sh-eco-head'>
            <h3 className='sh-eco-title'>{t('常用客户端')}</h3>
            <span className='sh-eco-count'>{t('全平台兼容')}</span>
          </div>
          <ul className='sh-eco-grid' role='list'>
            {CLIENTS.map((label) => {
              const Icon = CLIENT_ICONS[label];
              return (
                <li key={label}>
                  <Tooltip content={label}>
                    <span
                      className='sh-eco-item'
                      aria-label={label}
                      tabIndex={0}
                      role='button'
                    >
                      <span className='sh-eco-icon'>
                        <Icon size={18} strokeWidth={1.6} />
                      </span>
                      <span className='sh-eco-label'>{label}</span>
                    </span>
                  </Tooltip>
                </li>
              );
            })}
          </ul>
        </div>
      </div>
    </section>
  );
};

export default ProviderConstellation;
