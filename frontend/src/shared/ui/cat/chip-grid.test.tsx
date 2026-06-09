import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { ChipGrid } from "./chip-grid"

describe("ChipGrid", () => {
  it("marks active option with aria-pressed", () => {
    render(
      <ChipGrid
        value="a"
        onChange={() => {}}
        options={[
          { value: "a", label: "A" },
          { value: "b", label: "B" },
        ]}
      />
    )

    expect(screen.getByRole("button", { name: "A" })).toHaveAttribute(
      "aria-pressed",
      "true"
    )
    expect(screen.getByRole("button", { name: "B" })).toHaveAttribute(
      "aria-pressed",
      "false"
    )
  })

  it("calls onChange when option is clicked", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()

    render(
      <ChipGrid
        value="a"
        onChange={onChange}
        options={[
          { value: "a", label: "A" },
          { value: "b", label: "B" },
        ]}
      />
    )

    await user.click(screen.getByRole("button", { name: "B" }))
    expect(onChange).toHaveBeenCalledWith("b")
  })
})
