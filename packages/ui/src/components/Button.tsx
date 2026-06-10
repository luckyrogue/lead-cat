import type { ButtonHTMLAttributes } from "react";
import { cn } from "../lib/cn";

type Variant = "primary" | "soft" | "ghost";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
}

const variants: Record<Variant, string> = {
  primary: "bg-brand-500 text-white shadow-pop hover:bg-brand-600 active:translate-y-0.5",
  soft: "bg-surface-muted text-brand-700 hover:bg-surface-sunk",
  ghost: "bg-transparent text-brand-600 hover:bg-surface-soft",
};

export function Button({ variant = "primary", className, ...props }: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center rounded-2xl px-5 py-2.5 text-sm font-semibold transition-all focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-300 disabled:opacity-50",
        variants[variant],
        className,
      )}
      {...props}
    />
  );
}
