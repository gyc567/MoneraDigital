const getDaysRemaining = (endDate) => {
  const end = new Date(endDate);
  const now = new Date();
  const diff = end.getTime() - now.getTime();
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24));
  return days > 0 ? days : 0;
};

const formatNumber = (num) => {
  if (typeof num === "string") {
    num = parseFloat(num);
  }
  return new Intl.NumberFormat("en-US", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 8,
  }).format(num);
};

module.exports = { getDaysRemaining, formatNumber };
