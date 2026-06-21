const TIMEZONE_KEYS = [
  { value: "Asia/Almaty", key: "almaty" },
  { value: "Asia/Tashkent", key: "tashkent" },
  { value: "Asia/Bishkek", key: "bishkek" },
  { value: "Europe/Moscow", key: "moscow" },
  { value: "Europe/Kyiv", key: "kyiv" },
  { value: "Europe/London", key: "london" },
  { value: "Asia/Dubai", key: "dubai" },
  { value: "Asia/Istanbul", key: "istanbul" },
  { value: "America/New_York", key: "newYork" },
  { value: "UTC", key: "utc" },
] as const

type TFn = (key: string) => string

export function getTimezoneOptions(t: TFn) {
  return TIMEZONE_KEYS.map(({ value, key }) => ({
    value,
    label: t(`timezones.${key}`),
  }))
}

export function getTimezoneOptionsWithEmpty(t: TFn, emptyLabelKey: string) {
  return [{ value: "", label: t(emptyLabelKey) }, ...getTimezoneOptions(t)]
}
