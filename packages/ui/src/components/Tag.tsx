import type { HTMLAttributes } from "react";
import { cn } from "../lib/cn";

export type TagProps = HTMLAttributes<HTMLSpanElement>;

export function Tag({ className, ...props }: TagProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border-2 border-peach-200 bg-cream-100 px-3 py-1 text-sm font-semibold text-kitty-600",
        className,
      )}
      {...props}
    />
  );
}
