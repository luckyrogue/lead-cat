import type { SVGProps } from "react"

export function CatFace(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 96 96"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      <path d="M20 18 L34 38 L14 40 Z" fill="#E0995C" />
      <path d="M76 18 L62 38 L82 40 Z" fill="#E0995C" />
      <path d="M23 22 L31 36 L19 37 Z" fill="#FFB199" />
      <path d="M73 22 L65 36 L77 37 Z" fill="#FFB199" />
      <ellipse cx="48" cy="52" rx="34" ry="30" fill="#EDB57E" />
      <ellipse cx="48" cy="60" rx="22" ry="18" fill="#FBF1E6" />
      <circle cx="35" cy="48" r="5.5" fill="#4A3526" />
      <circle cx="61" cy="48" r="5.5" fill="#4A3526" />
      <circle cx="37" cy="46" r="1.8" fill="#fff" />
      <circle cx="63" cy="46" r="1.8" fill="#fff" />
      <path
        d="M44 60 Q48 64 52 60"
        stroke="#F2603F"
        strokeWidth="3"
        strokeLinecap="round"
        fill="none"
      />
      <path d="M44 58 L52 58 L48 62 Z" fill="#F2603F" />
      <g stroke="#C07A45" strokeWidth="2" strokeLinecap="round">
        <path d="M30 60 L12 56" />
        <path d="M30 64 L13 65" />
        <path d="M66 60 L84 56" />
        <path d="M66 64 L83 65" />
      </g>
    </svg>
  )
}
