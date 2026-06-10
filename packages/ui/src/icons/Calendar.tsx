import type { IconProps } from "./types";

export function Calendar(props: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      <rect x="3" y="5" width="18" height="16" rx="5" />
      <path d="M3 10h18" />
      <path d="M8 3v3M16 3v3" />
      <circle cx="8.5" cy="15" r="1.2" fill="currentColor" stroke="none" />
      <circle cx="15.5" cy="15" r="1.2" fill="currentColor" stroke="none" />
    </svg>
  );
}
