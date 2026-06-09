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

import jsQR from 'jsqr';

export const isLegacyQRCodeImageValue = (value) => {
  const text = String(value || '').trim();
  return (
    text.startsWith('data:image/') ||
    /^https?:\/\/.+\.(png|jpe?g|webp|gif)(\?.*)?(#.*)?$/i.test(text)
  );
};

export const decodeQRCodeImage = (file) =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const image = new Image();
      image.onload = () => {
        const canvas = document.createElement('canvas');
        canvas.width = image.naturalWidth || image.width;
        canvas.height = image.naturalHeight || image.height;
        const context = canvas.getContext('2d');
        if (!context) {
          reject(new Error('canvas unavailable'));
          return;
        }
        context.drawImage(image, 0, 0, canvas.width, canvas.height);
        const imageData = context.getImageData(
          0,
          0,
          canvas.width,
          canvas.height,
        );
        const qr = jsQR(imageData.data, imageData.width, imageData.height);
        if (!qr?.data) {
          reject(new Error('qr not found'));
          return;
        }
        resolve(qr.data);
      };
      image.onerror = () => reject(new Error('image load failed'));
      image.src = String(reader.result || '');
    };
    reader.onerror = () => reject(new Error('file read failed'));
    reader.readAsDataURL(file);
  });
