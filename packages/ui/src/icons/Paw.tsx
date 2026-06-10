import type { IconProps } from "./types";

export function Paw(props: IconProps) {
  return (
    <svg
      viewBox="0 0 64 64"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      <ellipse cx="32" cy="44" rx="15" ry="13" />
      <ellipse cx="14" cy="30" rx="6.5" ry="8" />
      <ellipse cx="50" cy="30" rx="6.5" ry="8" />
      <ellipse cx="24" cy="18" rx="6" ry="7.5" />
      <ellipse cx="40" cy="18" rx="6" ry="7.5" />
    </svg>
  );
}
