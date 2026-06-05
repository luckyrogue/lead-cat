import { useCallback, useEffect, useRef, useState } from "react"
import { useNavigate, useSearch } from "@tanstack/react-router"

import { DEFAULT_LIST_PAGE } from "@/shared/api/list-params"
import {
  areSearchParamsEqual,
  buildListSearchParams,
  parseListPage,
  parseListQuery,
} from "@/shared/lib/list-url-params"
import { useDebouncedValue } from "@/shared/lib/use-debounced-value"

type UseListUrlStateValues<TFilter> = {
  q: string
  page: number
  filter: TFilter
}

type UseListUrlStateOptions<TFilter> = {
  readFilter?: (searchParams: URLSearchParams) => TFilter
  buildSearchParams?: (
    values: UseListUrlStateValues<TFilter>
  ) => URLSearchParams
}

function toSearchRecord(params: URLSearchParams): Record<string, string> {
  const out: Record<string, string> = {}
  params.forEach((v, k) => {
    out[k] = v
  })
  return out
}

export function useListUrlState<TFilter = undefined>({
  readFilter,
  buildSearchParams,
}: UseListUrlStateOptions<TFilter> = {}) {
  const search = useSearch({ strict: false }) as Record<string, unknown>
  const navigate = useNavigate()
  const searchParams = new URLSearchParams()
  for (const [k, v] of Object.entries(search)) {
    if (v != null && v !== "") searchParams.set(k, String(v))
  }

  const [searchInput, setSearchInput] = useState(() =>
    parseListQuery(searchParams)
  )
  const debouncedSearch = useDebouncedValue(searchInput)
  const prevDebouncedRef = useRef(debouncedSearch)

  const page = parseListPage(searchParams)
  const filter = readFilter ? readFilter(searchParams) : (undefined as TFilter)

  useEffect(() => {
    setSearchInput(parseListQuery(searchParams))
  }, [searchParams.toString()])

  const writeUrl = useCallback(
    (values: UseListUrlStateValues<TFilter>) => {
      const params =
        buildSearchParams?.(values) ??
        buildListSearchParams({ q: values.q, page: values.page })
      if (areSearchParamsEqual(params, searchParams)) return
      void navigate({
        search: toSearchRecord(params) as never,
        replace: true,
      })
    },
    [buildSearchParams, navigate, searchParams]
  )

  useEffect(() => {
    const qChanged = prevDebouncedRef.current !== debouncedSearch
    prevDebouncedRef.current = debouncedSearch
    writeUrl({
      q: debouncedSearch,
      filter,
      page: qChanged ? DEFAULT_LIST_PAGE : page,
    })
  }, [debouncedSearch, filter, page, writeUrl])

  const setPage = useCallback(
    (nextPage: number) => {
      writeUrl({ q: debouncedSearch, filter, page: nextPage })
    },
    [debouncedSearch, filter, writeUrl]
  )

  const setFilter = useCallback(
    (nextFilter: TFilter) => {
      writeUrl({
        q: debouncedSearch,
        filter: nextFilter,
        page: DEFAULT_LIST_PAGE,
      })
    },
    [debouncedSearch, writeUrl]
  )

  return {
    search: searchInput,
    setSearch: setSearchInput,
    debouncedSearch,
    page,
    setPage,
    filter,
    setFilter,
  }
}
