import type { ReactNode } from "react";

/** Standard admin page header: title, subtitle and right-side actions. */
export function PageHead({ title, subtitle, actions }: { title: string; subtitle?: string; actions?: ReactNode }) {
  return (
    <div className="admin-pagehead">
      <div className="admin-pagehead__main">
        <h1 className="admin-h1">{title}</h1>
        {subtitle && <p className="admin-sub">{subtitle}</p>}
      </div>
      {actions && <div className="admin-pagehead__actions">{actions}</div>}
    </div>
  );
}
