import type { ElementType, HTMLAttributes, ReactNode } from 'react';

type PrimitiveTone = 'neutral' | 'info' | 'success' | 'error';

type PrimitiveProps<T extends ElementType> = {
  as?: T;
  className?: string;
  children?: ReactNode;
} & Omit<HTMLAttributes<HTMLElement>, 'className' | 'children'>;

type EmptyStateProps = {
  eyebrow?: string;
  title: string;
  description?: string;
  className?: string;
};

export function PageContainer({ as, className, children, ...props }: PrimitiveProps<ElementType>) {
  return renderPrimitive(as ?? 'div', joinClasses('foundation-page-container', className), children, props);
}

export function SectionCard({ as, className, children, ...props }: PrimitiveProps<ElementType>) {
  return renderPrimitive(as ?? 'section', joinClasses('card foundation-section-card', className), children, props);
}

export function Badge({ className, tone = 'neutral', children, ...props }: PrimitiveProps<'span'> & { tone?: PrimitiveTone }) {
  return (
    <span className={joinClasses('badge', tone, 'foundation-badge', className)} {...props}>
      {children}
    </span>
  );
}

export function InlineNotice({ as, className, children, ...props }: PrimitiveProps<ElementType>) {
  return renderPrimitive(as ?? 'div', joinClasses('foundation-inline-notice', className), children, props);
}

export function EmptyState({ eyebrow, title, description, className }: EmptyStateProps) {
  return (
    <section className={joinClasses('foundation-empty-state empty-state', className)}>
      {eyebrow ? <span className="hero-kicker">{eyebrow}</span> : null}
      <strong>{title}</strong>
      {description ? <span>{description}</span> : null}
    </section>
  );
}

export function ActionBar({ as, className, children, ...props }: PrimitiveProps<ElementType>) {
  return renderPrimitive(as ?? 'div', joinClasses('foundation-action-bar', className), children, props);
}

export function SummaryTile({ as, className, children, ...props }: PrimitiveProps<ElementType>) {
  return renderPrimitive(as ?? 'article', joinClasses('summary-tile foundation-summary-tile', className), children, props);
}

export function ContentSplit({ as, className, children, ...props }: PrimitiveProps<ElementType>) {
  return renderPrimitive(as ?? 'div', joinClasses('foundation-content-split', className), children, props);
}

function renderPrimitive(
  Component: ElementType,
  className: string,
  children: ReactNode,
  props: Omit<HTMLAttributes<HTMLElement>, 'className' | 'children'>,
) {
  return (
    <Component className={className} {...props}>
      {children}
    </Component>
  );
}

function joinClasses(...values: Array<string | undefined>) {
  return values.filter(Boolean).join(' ');
}
