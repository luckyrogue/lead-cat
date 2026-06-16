import type { Locale } from "~/shared/i18n/types"
import { en } from "~/shared/i18n/dictionaries/en"
import { kk } from "~/shared/i18n/dictionaries/kk"
import { ru } from "~/shared/i18n/dictionaries/ru"

export const dictionaries: Record<Locale, typeof en> = { en, ru, kk }
