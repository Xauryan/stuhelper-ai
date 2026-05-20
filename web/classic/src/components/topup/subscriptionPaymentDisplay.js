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
export function calculateSubscriptionPayAmount(priceAmount, unitPrice) {
  const price = Number(priceAmount);
  const priceRatio = Number(unitPrice);
  if (
    !Number.isFinite(price) ||
    price < 0 ||
    !Number.isFinite(priceRatio) ||
    priceRatio <= 0
  ) {
    return null;
  }
  return Math.ceil(price * priceRatio * 100 - 1e-9) / 100;
}

export function formatSubscriptionPayAmount({
  priceAmount,
  symbol,
  rate,
  unitPrice,
}) {
  const amount = calculateSubscriptionPayAmount(priceAmount, unitPrice);
  if (amount !== null) {
    return `¥${amount.toFixed(2)}`;
  }

  const price = Number(priceAmount);
  const currencyRate = Number(rate);
  const convertedPrice =
    (Number.isFinite(price) ? price : 0) *
    (Number.isFinite(currencyRate) ? currencyRate : 1);
  const displayPrice = convertedPrice.toFixed(
    Number.isInteger(convertedPrice) ? 0 : 2,
  );
  return `${symbol || '$'}${displayPrice}`;
}
