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

import React, { useMemo } from 'react';
import { useActualTheme } from '../../../context/Theme';

const ThemeToggle = ({ theme, onThemeToggle, t }) => {
  const actualTheme = useActualTheme();

  const nextTheme = actualTheme === 'dark' ? 'light' : 'dark';
  const ariaLabel = useMemo(
    () =>
      `${t('切换主题')}：${
        nextTheme === 'dark' ? t('深色模式') : t('浅色模式')
      }`,
    [nextTheme, t],
  );

  return (
    <button
      type='button'
      className={`classic-theme-switch classic-theme-switch--${actualTheme}`}
      data-theme-mode={theme}
      aria-label={ariaLabel}
      aria-pressed={actualTheme === 'dark'}
      title={ariaLabel}
      onClick={() => onThemeToggle(nextTheme)}
    >
      <span
        className='classic-theme-switch__sky classic-theme-switch__sky--light'
        aria-hidden='true'
      />
      <span
        className='classic-theme-switch__sky classic-theme-switch__sky--dark'
        aria-hidden='true'
      />
      <span className='classic-theme-switch__stars' aria-hidden='true'>
        <span className='classic-theme-switch__star classic-theme-switch__star--1' />
        <span className='classic-theme-switch__star classic-theme-switch__star--2' />
        <span className='classic-theme-switch__star classic-theme-switch__star--3' />
        <span className='classic-theme-switch__star classic-theme-switch__star--4' />
      </span>
      <span className='classic-theme-switch__clouds' aria-hidden='true'>
        <span className='classic-theme-switch__cloud classic-theme-switch__cloud--1' />
        <span className='classic-theme-switch__cloud classic-theme-switch__cloud--2' />
        <span className='classic-theme-switch__cloud classic-theme-switch__cloud--3' />
      </span>
      <span className='classic-theme-switch__handle' aria-hidden='true'>
        <span className='classic-theme-switch__crater classic-theme-switch__crater--1' />
        <span className='classic-theme-switch__crater classic-theme-switch__crater--2' />
        <span className='classic-theme-switch__crater classic-theme-switch__crater--3' />
      </span>
    </button>
  );
};

export default ThemeToggle;
