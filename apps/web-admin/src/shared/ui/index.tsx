import { useEffect, type ReactNode } from "react";

export function Spinner({ label }: { label?: string }) {
  return (
    <span className="row" style={{ gap: 8 }}>
      <span className="spinner" aria-hidden /> {label && <span className="muted">{label}</span>}
    </span>
  );
}

export function Notice({
  kind,
  children,
}: {
  kind: "danger" | "success" | "info";
  children: ReactNode;
}) {
  return (
    <div className={`notice notice-${kind}`} role={kind === "danger" ? "alert" : undefined}>
      {children}
    </div>
  );
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>;
}

/** StepDots — wizard/onboarding progress, e.g. ●──●──○ (current of total). */
/** ProgressRail — vertical step list for the two-column welcome/onboarding layout.
 *  `current` is 1-based; markers derive from it (done ✓ / current ring / upcoming). */
/** SuccessBurst — light gamified completion: a checklist of done + next items. */
export function Modal({
  title,
  onClose,
  children,
  wide,
}: {
  title?: string;
  onClose: () => void;
  children: ReactNode;
  wide?: boolean;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className={wide ? undefined : "modal"} onClick={(e) => e.stopPropagation()}>
        {title && <h2>{title}</h2>}
        {children}
      </div>
    </div>
  );
}
