import type { ButtonHTMLAttributes } from "react";
import { cn } from "../lib/cn";

type Variant = "primary" | "secondary" | "soft" | "sunny" | "ghost";
type Size = "sm" | "md" | "lg";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
}

const variants: Record<Variant, string> = {
  primary:
    "bg-coral-500 text-white shadow-pop hover:bg-coral-600 active:translate-y-1.5 active:shadow-none",
  secondary:
    "bg-peach-200 text-kitty-700 shadow-softer hover:bg-peach-300 active:translate-y-0.5",
  soft: "bg-peach-200 text-kitty-700 shadow-softer hover:bg-peach-300 active:translate-y-0.5",
  sunny:
    "bg-sunny-400 text-kitty-800 shadow-pop-sunny hover:bg-sunny-500 active:translate-y-1.5 active:shadow-none",
  ghost: "bg-transparent text-coral-600 hover:bg-coral-50 active:translate-y-0.5",
};

const sizes: Record<Size, string> = {
  sm: "px-4 py-2 text-sm",
  md: "px-6 py-3 text-base",
  lg: "px-8 py-4 text-lg",
};

export function Button({
  variant = "primary",
  size = "md",
  className,
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex select-none items-center justify-center gap-2 rounded-full font-bold tracking-wide transition-all duration-200 ease-spring hover:scale-105 focus:outline-none focus-visible:ring-4 focus-visible:ring-coral-200 disabled:pointer-events-none disabled:opacity-50",
        variants[variant],
        sizes[size],
        className,
      )}
      {...props}
    />
  );
}
