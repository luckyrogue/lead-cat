import { cn } from "@/shared/lib/cn"

export const ICON_PATHS = {
  home: "M3 11.5 12 4l9 7.5M5.5 10v9.5h13V10",
  calendar: "M4 6.5h16v14H4zM4 10h16M8 3.5v4M16 3.5v4",
  clock: "M12 7.5v5l3 2M12 3.5a8.5 8.5 0 100 17 8.5 8.5 0 000-17z",
  bolt: "M13 3 5 13.5h6L11 21l8-10.5h-6L13 3z",
  user: "M12 12.5a4 4 0 100-8 4 4 0 000 8zM5 20c.7-3.4 3.6-5.5 7-5.5s6.3 2.1 7 5.5",
  users:
    "M9 11.5a3.4 3.4 0 100-6.8 3.4 3.4 0 000 6.8zM3.5 19c.6-2.9 3-4.6 5.5-4.6S14 16.1 14.5 19M16 5.2a3.2 3.2 0 011.4 6M17 14.6c2 .5 3.6 2 4 4.4",
  plus: "M12 5v14M5 12h14",
  chevR: "M9 5l7 7-7 7",
  chevL: "M15 5l-7 7 7 7",
  chevD: "M5 9l7 7 7-7",
  x: "M6 6l12 12M18 6L6 18",
  globe:
    "M12 3.5a8.5 8.5 0 100 17 8.5 8.5 0 000-17zM3.5 12h17M12 3.5c2.5 2.3 2.5 14.2 0 17M12 3.5c-2.5 2.3-2.5 14.2 0 17",
  check: "M5 12.5l4.5 4.5L19 7",
  trash: "M5 7h14M9 7V5h6v2M7 7l1 13h8l1-13",
  pencil: "M4 20h4L19 9l-4-4L4 16v4zM14 6l4 4",
  link: "M10 14a4 4 0 005.7 0l3-3a4 4 0 00-5.7-5.7L11 7M14 10a4 4 0 00-5.7 0l-3 3A4 4 0 009 18.7L11 17",
  bell: "M6 9a6 6 0 1112 0c0 6 2 7 2 7H4s2-1 2-7zM10 21h4",
  search: "M11 4.5a6.5 6.5 0 104.6 11.1A6.5 6.5 0 0011 4.5zM16 16l4 4",
  arrowR: "M5 12h14M13 6l6 6-6 6",
  settings:
    "M12 9.5a2.5 2.5 0 100 5 2.5 2.5 0 000-5zM12 3l1.3 2.4 2.7-.5.5 2.7L20 11l-2.5 1.3.5 2.7-2.7.5L12 21l-1.3-2.5-2.7.5-.5-2.7L4 11l2.5-1.3-.5-2.7 2.7.5L12 3z",
  shield: "M12 3.5l7 2.5v5c0 5-3.2 8-7 9.5-3.8-1.5-7-4.5-7-9.5v-5l7-2.5z",
  sparkle: "M12 4l1.6 4.4L18 10l-4.4 1.6L12 16l-1.6-4.4L6 10l4.4-1.6L12 4z",
  repeat: "M4 9V7a3 3 0 013-3h10l-2.5-2.5M20 15v2a3 3 0 01-3 3H7l2.5 2.5",
  doc: "M6 3.5h8l4 4v13H6zM14 3.5v4h4",
  flame:
    "M12 3c1 3-2 4-2 7a2 2 0 104 0c0-1-.5-1.5-.5-1.5C15 11 17 12 17 15a5 5 0 11-10 0c0-4 5-7 5-12z",
} as const

export type IconName = keyof typeof ICON_PATHS

type IconProps = {
  name: IconName
  size?: number
  sw?: number
  fill?: boolean
  className?: string
}

export function CatIcon({
  name,
  size = 22,
  sw = 1.9,
  fill = false,
  className,
}: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      className={cn("shrink-0", className)}
      aria-hidden
    >
      <path
        d={ICON_PATHS[name] ?? ""}
        stroke={fill ? "none" : "currentColor"}
        fill={fill ? "currentColor" : "none"}
        strokeWidth={sw}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}
