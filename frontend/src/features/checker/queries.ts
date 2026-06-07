import { useMutation } from "@tanstack/react-query"
import {
  fetchFreeSlots,
  type FreeSlotsParams,
} from "@/entities/meeting/scheduling-api"
import { useTmaApp } from "@/shared/tma/context"

export function useFreeSlots() {
  const { lang } = useTmaApp()
  return useMutation({
    mutationFn: (params: FreeSlotsParams) => fetchFreeSlots(params, lang),
  })
}
