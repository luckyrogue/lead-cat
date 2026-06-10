import { apiFetch } from "~/shared/api/client"
import type { Employee } from "~/entities/employee/types"

export async function searchEmployees(q: string): Promise<Employee[]> {
  const res = await apiFetch<{ employees: Employee[] }>("/api/miniapp/employees", {
    params: { q },
  })
  return res.employees ?? []
}
