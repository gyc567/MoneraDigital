import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Format number with thousands separator and clean trailing zeros
 * @param num Number to format
 * @param precision Maximum decimal places (default: 7)
 * @param minFractionDigits Minimum decimal places
 * @param maxFractionDigits Maximum decimal places
 * @returns Formatted number string
 */
export function formatNumber(
  num: string | number,
  precision?: number,
  minFractionDigits?: number,
  maxFractionDigits?: number
): string {
  if (num === '' || num === null || num === undefined) {
    return '0';
  }

  if (typeof num === 'string') {
    const parsed = parseFloat(num);
    if (isNaN(parsed)) return '0';
    num = parsed;
  }

  if (isNaN(num) || !isFinite(num)) {
    return '0';
  }

  // Handle precision parameters
  const localeOptions: Intl.NumberFormatOptions = {
    minimumFractionDigits: minFractionDigits !== undefined ? minFractionDigits : 0,
    maximumFractionDigits: maxFractionDigits !== undefined ? maxFractionDigits : (precision !== undefined ? precision : 7),
  };

  const formatted = new Intl.NumberFormat("en-US", localeOptions).format(num);

  // Clean up trailing zeros for better display
  if (precision !== undefined || minFractionDigits === undefined) {
    return formatted.replace(/(\.\d*?[1-9])0+$/g, '$1').replace(/\.$/, '');
  }

  return formatted;
}
