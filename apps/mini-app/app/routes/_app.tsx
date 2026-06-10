import { Outlet } from "react-router"

export default function AppLayout() {
  return (
    <div className="mx-auto min-h-svh w-full max-w-md px-4 pt-6 pb-10">
      <Outlet />
    </div>
  )
}
