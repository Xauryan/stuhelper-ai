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

const SectionHeader = ({ eyebrow, title, description }) => {
  return (
    <header className='sh-section-header sh-reveal'>
      {eyebrow ? <span className='sh-eyebrow'>{eyebrow}</span> : null}
      <h2 className='sh-section-title'>{title}</h2>
      {description ? <p className='sh-section-desc'>{description}</p> : null}
    </header>
  );
};

export default SectionHeader;
