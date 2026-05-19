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

import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { API, showError, copy, showSuccess } from '../../helpers';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { API_ENDPOINTS } from '../../constants/common.constant';
import { StatusContext } from '../../context/Status';
import { useActualTheme } from '../../context/Theme';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import NoticeModal from '../../components/layout/NoticeModal';
import QuantumHome from './QuantumHome';

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const actualTheme = useActualTheme();
  const isMobile = useIsMobile();

  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageRawContent, setHomePageRawContent] = useState('');
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);
  const [endpointIndex, setEndpointIndex] = useState(0);

  const iframeRef = useRef(null);

  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;
  const docsLink = statusState?.status?.docs_link || '';
  const version = statusState?.status?.version || '';
  const serverAddress =
    statusState?.status?.server_address || `${window.location.origin}`;
  const endpointItems = useMemo(
    () => API_ENDPOINTS.map((e) => ({ value: e })),
    [],
  );
  const isChinese = i18n.language.startsWith('zh');

  const renderHomeContent = (raw) => {
    if (!raw) return '';
    if (raw.startsWith('https://')) return raw;
    return marked.parse(raw);
  };

  const postIframePreferences = useCallback(() => {
    const win = iframeRef.current?.contentWindow;
    if (!win) return;
    win.postMessage({ themeMode: actualTheme }, '*');
    win.postMessage({ lang: i18n.language }, '*');
  }, [actualTheme, i18n.language]);

  const displayHomePageContent = async (isMounted = () => true) => {
    const cached =
      localStorage.getItem('home_page_content_raw') ??
      localStorage.getItem('home_page_content') ??
      '';
    if (!isMounted()) return;
    setHomePageRawContent(cached);
    setHomePageContent(renderHomeContent(cached));
    try {
      const res = await API.get('/api/home_page_content');
      if (!isMounted()) return;
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      const raw = data || '';
      setHomePageRawContent(raw);
      setHomePageContent(renderHomeContent(raw));
      localStorage.setItem('home_page_content_raw', raw);
    } catch (error) {
      if (isMounted()) console.error('加载首页内容失败:', error);
    } finally {
      if (isMounted()) setHomePageContentLoaded(true);
    }
  };

  const handleCopyBaseURL = async () => {
    const ok = await copy(serverAddress);
    if (ok) {
      showSuccess(t('已复制到剪切板'));
    }
  };

  useEffect(() => {
    let mounted = true;
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          if (!mounted) return;
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          if (mounted) console.error('获取公告失败:', error);
        }
      }
    };
    checkNoticeAndShow();
    return () => {
      mounted = false;
    };
  }, []);

  useEffect(() => {
    let mounted = true;
    displayHomePageContent(() => mounted).then();
    return () => {
      mounted = false;
    };
  }, []);

  useEffect(() => {
    const timer = setInterval(() => {
      setEndpointIndex((prev) => (prev + 1) % endpointItems.length);
    }, 3000);
    return () => clearInterval(timer);
  }, [endpointItems.length]);

  useEffect(() => {
    if (homePageRawContent.startsWith('https://')) {
      postIframePreferences();
    }
  }, [homePageRawContent, postIframePreferences]);

  return (
    <div className='w-full overflow-x-hidden'>
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile}
      />
      {homePageContentLoaded && homePageRawContent === '' ? (
        <QuantumHome
          t={t}
          isChinese={isChinese}
          isMobile={isMobile}
          serverAddress={serverAddress}
          endpointItems={endpointItems}
          endpointIndex={endpointIndex}
          onSelectEndpoint={setEndpointIndex}
          onCopyBaseURL={handleCopyBaseURL}
          isDemoSiteMode={isDemoSiteMode}
          docsLink={docsLink}
          version={version}
        />
      ) : (
        <div className='overflow-x-hidden w-full'>
          {homePageRawContent.startsWith('https://') ? (
            <iframe
              ref={iframeRef}
              src={homePageRawContent}
              className='w-full h-screen border-none'
              onLoad={postIframePreferences}
              title={t('Custom homepage')}
            />
          ) : (
            <div
              className='mt-[60px]'
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            />
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
