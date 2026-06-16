import { SITE_NAME } from "./site"

const LOGO_MARK_SRC = "/logo-mark.svg"

type LeadCatLogoProps = {
  subtitle?: string
  variant?: "lockup" | "stacked" | "mark"
  className?: string
  markClassName?: string
  textClassName?: string
}

function logoMarkSize(variant: LeadCatLogoProps["variant"]) {
  if (variant === "stacked") {
    return { className: "size-14", width: 56, height: 56 }
  }

  return { className: "size-9", width: 36, height: 36 }
}

export function LeadCatLogo({
  subtitle,
  variant = "lockup",
  className = "",
  markClassName = "",
  textClassName = "",
}: LeadCatLogoProps) {
  const { className: sizeClassName, width, height } = logoMarkSize(variant)

  const mark = (
    <img
      src={LOGO_MARK_SRC}
      alt=""
      width={width}
      height={height}
      className={`shrink-0 rounded-2xl shadow-[0_18px_40px_-22px_oklch(0.64_0.2_28_/_0.55)] ${sizeClassName} ${markClassName}`}
    />
  )

  if (variant === "mark") {
    return <span className={`inline-flex ${className}`}>{mark}</span>
  }

  if (variant === "stacked") {
    return (
      <div className={`flex flex-col items-center gap-3 ${className}`}>
        {mark}
        <div className="text-center">
          <p
            className={`text-xl font-semibold tracking-tight text-kitty-800 ${textClassName}`}
          >
            {SITE_NAME}
          </p>
          {subtitle ? (
            <p className="text-xs font-medium tracking-[0.16em] text-muted-foreground uppercase">
              {subtitle}
            </p>
          ) : null}
        </div>
      </div>
    )
  }

  return (
    <span className={`inline-flex items-center gap-2 font-bold text-kitty-800 ${className}`}>
      {mark}
      <span className={textClassName}>{SITE_NAME}</span>
    </span>
  )
}
