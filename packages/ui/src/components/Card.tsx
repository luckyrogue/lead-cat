import type { HTMLAttributes } from "react";
import { cn } from "../lib/cn";

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  bordered?: boolean;
}

export function Card({ bordered = false, className, ...props }: CardProps) {
  return (
    <div
      className={cn(
        "rounded-3xl bg-white/80 p-7 shadow-soft backdrop-blur-sm",
        bordered && "border-2 border-peach-200",
        className,
      )}
      {...props}
    />
  );
}
