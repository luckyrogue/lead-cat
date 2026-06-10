import type { IconProps } from "./types";

export function Heart(props: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      <path d="M12 21s-7.5-4.7-10-9.3C.4 8.6 1.8 5 5.2 5c2 0 3.3 1.1 4.1 2.3.7 1 .7 1 .7 1s0 0 .7-1C11.5 6.1 12.8 5 14.8 5c3.4 0 4.8 3.6 3.2 6.7C19.5 16.3 12 21 12 21z" />
    </svg>
  );
}
