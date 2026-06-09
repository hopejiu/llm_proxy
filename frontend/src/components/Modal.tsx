import { useEffect, useRef, type ReactNode } from "react";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  footer?: ReactNode;
  className?: string;
}

export default function Modal({ open, onClose, title, children, footer, className = "" }: ModalProps) {
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => { if (e.key === "Escape") onClose(); };
    globalThis.addEventListener("keydown", handler);
    contentRef.current?.focus();
    return () => globalThis.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="modal-overlay" onClick={onClose} role="dialog" aria-modal="true" aria-label={title}>
      <div ref={contentRef} tabIndex={-1} className={`modal-content ${className}`} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 className="text-base font-semibold text-[#1E1B2E]">{title}</h2>
          <button onClick={onClose} className="text-[#9C94B0] hover:text-[#6B6580] transition-colors" aria-label="关闭">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-footer">{footer}</div>}
      </div>
    </div>
  );
}
