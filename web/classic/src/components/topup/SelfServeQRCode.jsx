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
import PropTypes from 'prop-types';
import { QRCodeSVG } from 'qrcode.react';
import { isLegacyQRCodeImageValue } from './qrCodeUtils';

const SelfServeQRCode = ({ value, alt = '', size = 220, level = 'M' }) => {
  const text = String(value || '').trim();
  const [imageFailed, setImageFailed] = useState(false);

  useEffect(() => {
    setImageFailed(false);
  }, [text]);

  if (!text) {
    return null;
  }

  if (isLegacyQRCodeImageValue(text) && !imageFailed) {
    return (
      <img
        src={text}
        alt={alt}
        style={{ width: size, height: size, objectFit: 'contain' }}
        onError={() => setImageFailed(true)}
      />
    );
  }

  return <QRCodeSVG value={text} size={size} level={level} />;
};

SelfServeQRCode.propTypes = {
  value: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
  alt: PropTypes.string,
  size: PropTypes.number,
  level: PropTypes.oneOf(['L', 'M', 'Q', 'H']),
};

export default SelfServeQRCode;
