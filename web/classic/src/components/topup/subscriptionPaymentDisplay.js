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
