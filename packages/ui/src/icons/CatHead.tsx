import type { IconProps } from "./types";

export function CatHead(props: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      <path d="M5 3.5c.2-.5.9-.5 1.1 0l1.6 4.1A8.4 8.4 0 0 1 12 6.7c1.5 0 2.9.3 4.3.9l1.6-4.1c.2-.5.9-.5 1.1 0l1.3 3.4c.5 1.3.5 2.7 0 4 .8 1 1.3 2.3 1.3 3.7 0 3.6-3.6 6.4-8 6.4-3.6 0-7.6-2.5-8-6 0-.1 0-.3 0-.4 0-1.4.5-2.7 1.3-3.7-.5-1.3-.5-2.7 0-4L5 3.5z" />
      <circle cx="9" cy="13.5" r="1.2" fill="#fff" />
      <circle cx="15" cy="13.5" r="1.2" fill="#fff" />
      <path
        d="M11 16.5q1 .8 2 0"
        stroke="#fff"
        strokeWidth="1.1"
        strokeLinecap="round"
        fill="none"
      />
    </svg>
  );
}
