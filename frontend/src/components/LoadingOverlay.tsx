import LoadingSpinner from "./LoadingSpinner";

export default function LoadingOverlay({ text = "加载中..." }: { text?: string }) {
  return (
    <div className="modal-overlay">
      <LoadingSpinner text={text} className="py-0" />
    </div>
  );
}
