import type { HTMLAttributes } from "react";
import { cn } from "../lib/cn";

type Tone = "coral" | "peach" | "sunny" | "kitty";

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: Tone;
}

const tones: Record<Tone, string> = {
  coral: "bg-coral-100 text-coral-700",
  peach: "bg-peach-100 text-kitty-700",
  sunny: "bg-sunny-100 text-sunny-600",
  kitty: "bg-kitty-100 text-kitty-700",
};

export function Badge({ tone = "coral", className, ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-bold uppercase tracking-wider",
        tones[tone],
        className,
      )}
      {...props}
    />
  );
}
