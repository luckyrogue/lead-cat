import { LeadCatLogo } from "@leadcat/brand"

type BrandLogoProps = {
  subtitle?: string
}

export function BrandLogo({ subtitle }: BrandLogoProps) {
  return <LeadCatLogo variant="stacked" subtitle={subtitle} />
}
