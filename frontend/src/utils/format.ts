const numberFormat = new Intl.NumberFormat('en-IN');
const compactFormat = new Intl.NumberFormat('en-IN', { notation: 'compact', maximumFractionDigits: 1 });
const currencyFormat = new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 2 });

export function formatNumber(value: number): string {
  return numberFormat.format(Math.round(value));
}

export function formatCompact(value: number): string {
  return compactFormat.format(value);
}

export function formatCurrency(value: number): string {
  return currencyFormat.format(value);
}

export function formatMillis(value: number): string {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(2)} s`;
  }
  return `${value.toFixed(value < 10 ? 1 : 0)} ms`;
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function formatDuration(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = Math.floor(totalSeconds % 60);
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}

export function classNames(...values: (string | false | null | undefined)[]): string {
  return values.filter(Boolean).join(' ');
}

export function formatDate(value: string): string {
  return new Date(value).toLocaleDateString('en-IN', { day: '2-digit', month: 'short', year: 'numeric' });
}
