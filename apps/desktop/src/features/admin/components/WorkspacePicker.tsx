import { useEffect, useRef, useState } from "react";
import { initials } from "../lib";
import type { OwnerBusiness } from "../types";

const svgp = { viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 2, strokeLinecap: "round", strokeLinejoin: "round" } as const;
const ChevronsUpDown = () => (<svg {...svgp} width="16" height="16" aria-hidden><path d="m7 15 5 5 5-5" /><path d="m7 9 5-5 5 5" /></svg>);
const CheckIcon = () => (<svg {...svgp} width="16" height="16" aria-hidden><path d="M20 6 9 17l-5-5" /></svg>);

// The workspace (business) picker shown in the app topbar on admin screens. Lets
// the owner switch which workspace the admin screens act on. Workspaces map 1:1 to
// Selamnew organizations and are created on sign-in, so there is no "new team".
// Reuses the `bb-adshell__biz*` styles.
export function WorkspacePicker({
  businesses,
  bizId,
  onSelect,
}: {
  businesses: OwnerBusiness[];
  bizId: string | null;
  onSelect: (id: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  const selected = businesses.find((b) => b.id === bizId) ?? null;
  if (!selected) return null;

  return (
    <div className="bb-adshell__bizwrap" ref={ref}>
      <button
        type="button"
        className="bb-adshell__biz"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <span className="bb-adshell__bizic">{initials(selected.name)}</span>
        <span className="bb-adshell__bizname">{selected.name}</span>
        <span className="bb-adshell__bizcaret"><ChevronsUpDown /></span>
      </button>
      {open && (
        <div className="bb-adshell__bizmenu" role="menu">
          {businesses.map((b) => (
            <button
              key={b.id}
              type="button"
              role="menuitemradio"
              aria-checked={b.id === bizId}
              className={`bb-adshell__bizopt ${b.id === bizId ? "on" : ""}`}
              onClick={() => {
                onSelect(b.id);
                setOpen(false);
              }}
            >
              <span className="bb-adshell__bizic">{initials(b.name)}</span>
              <span className="bb-adshell__bizoptname">{b.name}</span>
              {b.id === bizId && <span className="bb-adshell__bizck"><CheckIcon /></span>}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
