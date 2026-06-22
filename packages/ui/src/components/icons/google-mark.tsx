import type { SVGProps } from "react"

export function GoogleMark(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path
        fill="#EA4335"
        d="M12 10.8v3.6h5.1c-.2 1.3-1.6 3.9-5.1 3.9-3.1 0-5.6-2.6-5.6-5.7s2.5-5.7 5.6-5.7c1.8 0 3 .8 3.6 1.4l2.5-2.4C16.6 3.9 14.5 3 12 3 6.9 3 2.8 7.1 2.8 12S6.9 21 12 21c5.2 0 8.7-3.7 8.7-8.8 0-.6-.1-1-.2-1.4H12z"
      />
    </svg>
  )
}
