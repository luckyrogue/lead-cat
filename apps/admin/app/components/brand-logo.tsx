import { Paw } from "@leadcat/ui"

type BrandLogoProps = {
  subtitle?: string
}

export function BrandLogo({ subtitle }: BrandLogoProps) {
  return (
    <div className="flex flex-col items-center gap-3">
      <span className="grid size-14 place-items-center rounded-[calc(var(--radius)*1.1)] bg-coral-400 text-white shadow-[0_18px_40px_-22px_oklch(0.64_0.2_28_/_0.55)]">
        <Paw className="size-7" />
      </span>
      <div className="text-center">
        <p className="text-xl font-semibold tracking-tight text-kitty-800">
          Lead Cat
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
