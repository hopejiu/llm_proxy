interface SkeletonProps {
  className?: string;
  variant?: "text" | "rect" | "circle";
}

export default function Skeleton({ className = "", variant = "text" }: SkeletonProps) {
  const base = "animate-pulse bg-slate-200";
  const shape = variant === "circle" ? "rounded-full" : variant === "rect" ? "rounded-xl" : "rounded";
  return <div className={`${base} ${shape} ${className}`} />;
}
