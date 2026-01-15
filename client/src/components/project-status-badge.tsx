import { Badge } from '@/components/ui/badge';
import { ProjectStatus } from '@/types/project';
import { CheckCircle2, Circle, Loader2, XCircle } from 'lucide-react';

interface ProjectStatusBadgeProps {
  status: ProjectStatus;
}

export function ProjectStatusBadge({ status }: ProjectStatusBadgeProps) {
  const configs: Record<ProjectStatus, {
    label: string;
    variant: 'secondary' | 'default' | 'destructive';
    icon: React.ComponentType<{ className?: string }>;
    animate: boolean;
    className?: string;
  }> = {
    inactive: {
      label: 'Inactive',
      variant: 'secondary',
      icon: Circle,
      animate: false,
    },
    cloning: {
      label: 'Cloning',
      variant: 'default',
      icon: Loader2,
      animate: true,
    },
    active: {
      label: 'Active',
      variant: 'default',
      icon: CheckCircle2,
      animate: false,
      className: 'bg-green-500 hover:bg-green-600',
    },
    error: {
      label: 'Error',
      variant: 'destructive',
      icon: XCircle,
      animate: false,
    },
  };

  const config = configs[status];
  const Icon = config.icon;

  return (
    <Badge variant={config.variant} className={config.className}>
      <Icon
        className={`mr-1 h-3 w-3 ${config.animate ? 'animate-spin' : ''}`}
      />
      {config.label}
    </Badge>
  );
}
