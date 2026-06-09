import { useMutation } from "@tanstack/react-query"
import type { Lang } from "@/entities/meeting/lib/format"
import {
  fetchFreeSlots,
  type FreeSlotsParams,
} from "@/entities/meeting/scheduling-api"

export function useFreeSlots(lang: Lang) {
  return useMutation({
    mutationFn: (params: FreeSlotsParams) => fetchFreeSlots(params, lang),
  })
}
