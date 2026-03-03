'use client';

import Link from 'next/link';
import { Plus, RefreshCw, Minus, Bug } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import type { Changelog, ChangelogCategory, ChangelogSection } from '@/lib/types/changelog';
import { cn } from '@/lib/utils';

const categoryConfig: Record<
  ChangelogCategory,
  { label: string; icon: React.ElementType; color: string; bgColor: string }
> = {
  added: {
    label: 'Added',
    icon: Plus,
    color: 'text-green-600 dark:text-green-400',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
  },
  updated: {
    label: 'Updated',
    icon: RefreshCw,
    color: 'text-blue-600 dark:text-blue-400',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
  },
  removed: {
    label: 'Removed',
    icon: Minus,
    color: 'text-red-600 dark:text-red-400',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
  },
  fixed: {
    label: 'Fixed',
    icon: Bug,
    color: 'text-orange-600 dark:text-orange-400',
    bgColor: 'bg-orange-100 dark:bg-orange-900/30',
  },
};

interface Props {
  changelog: Changelog;
}

export function ChangelogTimelineItem({ changelog }: Props) {
  return (
    <Card className="overflow-hidden">
      {/* Header */}
      <div className="border-b bg-muted/30 px-6 py-4">
        <div className="flex flex-wrap items-center gap-3">
          <Badge variant="secondary" className="font-mono text-sm">
            v{changelog.version}
          </Badge>
          <h2 className="text-xl font-semibold">{changelog.title}</h2>
        </div>
        {changelog.description && (
          <p className="mt-2 text-muted-foreground">{changelog.description}</p>
        )}
      </div>

      {/* Sections */}
      <div className="divide-y">
        {(changelog.sections || []).map((section) => (
          <ChangelogSectionDisplay key={section.id} section={section} />
        ))}
      </div>

      {/* Contributors */}
      {changelog.contributors && changelog.contributors.length > 0 && (
        <div className="border-t bg-muted/20 px-6 py-6">
          <h4 className="mb-5 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
            Contributors
          </h4>
          <div className="grid gap-4 sm:grid-cols-2">
            {changelog.contributors.map((contributor) => (
              <Link
                key={contributor.id}
                href={`/${contributor.user?.username || ''}`}
                className="group flex items-center gap-4 rounded-xl border bg-background p-4 transition-all hover:border-primary hover:bg-muted hover:shadow-md"
              >
                <Avatar className="h-12 w-12 ring-2 ring-background">
                  <AvatarImage src={contributor.user?.avatar_url} />
                  <AvatarFallback className="text-base font-semibold">
                    {contributor.user?.nama?.charAt(0) || '?'}
                  </AvatarFallback>
                </Avatar>
                <div className="flex flex-col min-w-0">
                  <span className="font-semibold group-hover:text-primary truncate">
                    {contributor.user?.nama || 'Unknown'}
                  </span>
                  <span className="text-sm text-muted-foreground truncate">
                    @{contributor.user?.username || 'unknown'}
                  </span>
                  <span className="mt-1 text-sm text-primary/80 font-medium">
                    {contributor.contribution}
                  </span>
                </div>
              </Link>
            ))}
          </div>
        </div>
      )}
    </Card>
  );
}

function ChangelogSectionDisplay({ section }: { section: ChangelogSection }) {
  const config = categoryConfig[section.category];
  const Icon = config.icon;

  return (
    <div className="px-6 py-5">
      {/* Section header */}
      <div className="mb-4 flex items-center gap-2.5">
        <div className={cn('rounded-lg p-2', config.bgColor)}>
          <Icon className={cn('h-4 w-4', config.color)} />
        </div>
        <h3 className={cn('text-lg font-semibold', config.color)}>{config.label}</h3>
      </div>

      {/* Section content */}
      <div className="space-y-4 pl-11">
        {(section.blocks || []).map((block) => (
          <ChangelogBlockRenderer key={block.id} block={block} />
        ))}
      </div>
    </div>
  );
}

function ChangelogBlockRenderer({ block }: { block: { id: string; block_type: string; payload: Record<string, unknown> } }) {
  const payload = block.payload || {};

  switch (block.block_type) {
    case 'text':
      return (
        <div
          className="prose prose-sm dark:prose-invert max-w-none prose-p:leading-relaxed prose-ul:my-2 prose-li:my-0"
          dangerouslySetInnerHTML={{ __html: (payload.content as string) || '' }}
        />
      );

    case 'image':
      return (
        <figure className="space-y-2">
          {payload.url && (
            <img
              src={payload.url as string}
              alt={(payload.caption as string) || 'Image'}
              className="rounded-lg max-w-full h-auto"
            />
          )}
          {payload.caption && (
            <figcaption className="text-sm text-muted-foreground text-center">
              {payload.caption as string}
            </figcaption>
          )}
        </figure>
      );

    case 'youtube':
      return (
        <div className="aspect-video overflow-hidden rounded-lg">
          <iframe
            src={`https://www.youtube.com/embed/${payload.video_id}`}
            title={(payload.title as string) || 'YouTube video'}
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
            allowFullScreen
            className="h-full w-full"
          />
        </div>
      );

    case 'button':
      return (
        <a
          href={payload.url as string}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          {(payload.text as string) || 'Link'}
        </a>
      );

    default:
      return null;
  }
}
