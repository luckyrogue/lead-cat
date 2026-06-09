import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { Segmented } from "./segmented"

describe("Segmented", () => {
  it("marks active tab with aria-selected", () => {
    render(
      <Segmented
        value="up"
        onChange={() => {}}
        options={[
          { value: "up", label: "Upcoming" },
          { value: "past", label: "Past" },
        ]}
      />
    )

    expect(screen.getByRole("tab", { name: "Upcoming" })).toHaveAttribute(
      "aria-selected",
      "true"
    )
    expect(screen.getByRole("tab", { name: "Past" })).toHaveAttribute(
      "aria-selected",
      "false"
    )
  })

  it("calls onChange when tab is clicked", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()

    render(
      <Segmented
        value="up"
        onChange={onChange}
        options={[
          { value: "up", label: "Upcoming" },
          { value: "past", label: "Past" },
        ]}
      />
    )

    await user.click(screen.getByRole("tab", { name: "Past" }))
    expect(onChange).toHaveBeenCalledWith("past")
  })
})
