import Modal from "./Modal";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmDialog({
  open, title, message, confirmText = "确认", cancelText = "取消",
  danger = false, onConfirm, onCancel,
}: ConfirmDialogProps) {
  return (
    <Modal open={open} onClose={onCancel} title={title}
      footer={
        <div className="flex gap-2 justify-end">
          <button onClick={onCancel} className="btn-secondary">{cancelText}</button>
          <button onClick={onConfirm} className={danger ? "btn-danger" : "btn-primary"}>{confirmText}</button>
        </div>
      }
    >
      <p className="text-sm text-[#6B6580]">{message}</p>
    </Modal>
  );
}
