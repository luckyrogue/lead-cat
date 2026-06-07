import { apiFetch } from "@/shared/api/client"
import type { Employee } from "@/entities/employee/types"

type EmployeeDTO = {
  id: string
  name: string
  email: string
  dept: string
  tg: boolean
}

export async function searchEmployees(
  q: string,
  signal?: AbortSignal
): Promise<Employee[]> {
  const data = await apiFetch<{ employees: EmployeeDTO[] }>("/tma/employees", {
    params: { q },
    signal,
  })
  return data.employees.map((e) => ({
    id: e.id,
    name: e.name,
    email: e.email,
    dept: e.dept,
    tg: e.tg,
  }))
}
