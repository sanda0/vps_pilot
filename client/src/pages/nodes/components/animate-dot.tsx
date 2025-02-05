interface AnimateDotProps {
  color: 'success' | 'danger' | 'warning' | 'info';
  size: number;
  extraClass?: string;
}

const colorClasses = {
  success: 'bg-green-500',
  danger: 'bg-red-500',
  warning: 'bg-yellow-500',
  info: 'bg-blue-500',
};

export default function AnimateDot({ color, size, extraClass = '' }: AnimateDotProps) {
  const colorClass = colorClasses[color] || 'bg-green-500';
  const sizeClass = `h-${size} w-${size}`;

  return (
    <span className={`relative flex ${sizeClass} ${extraClass}`}>
      <span className={`animate-ping absolute inline-flex h-full w-full rounded-full ${colorClass} opacity-75`}></span>
      <span className={`relative inline-flex rounded-full ${sizeClass} ${colorClass}`}></span>
    </span>
  );
}
