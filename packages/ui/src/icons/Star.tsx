import type { IconProps } from "./types";

export function Star(props: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      <path d="M12 2.5c.4 0 .8.25 1 .65l2.3 4.66 5.15.75c.95.14 1.33 1.3.64 1.97l-3.73 3.63.88 5.13c.16.94-.83 1.66-1.68 1.21L12 18.85l-4.6 2.42c-.85.45-1.84-.27-1.68-1.21l.88-5.13L2.87 11.3c-.69-.67-.31-1.83.64-1.97L8.66 7.81 11 3.15c.2-.4.6-.65 1-.65z" />
    </svg>
  );
}
