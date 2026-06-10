import type { IconProps } from "./types";

export function Sparkle(props: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      <path d="M12 0c.6 5.4 2 6.8 7.4 7.4C14 8 12.6 9.4 12 14.8 11.4 9.4 10 8 4.6 7.4 10 6.8 11.4 5.4 12 0z" />
      <path d="M19 14c.3 2.4.9 3 3.3 3.3-2.4.3-3 .9-3.3 3.3-.3-2.4-.9-3-3.3-3.3 2.4-.3 3-.9 3.3-3.3z" />
    </svg>
  );
}
