import type { ReactNode } from 'react';

interface ModalProps {
  title: string;
  onClose: () => void;
  children: ReactNode;
}

export function Modal({ children, onClose, title }: ModalProps) {
  return (
    <div className="modalBackdrop" role="presentation" onClick={onClose}>
      <section
        aria-modal="true"
        className="modal"
        role="dialog"
        aria-labelledby="modal-title"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modalHeader">
          <h2 id="modal-title">{title}</h2>
          <button type="button" className="iconButton" onClick={onClose} aria-label="Close detail">
            ×
          </button>
        </div>
        {children}
      </section>
    </div>
  );
}
