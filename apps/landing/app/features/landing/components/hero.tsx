import { Badge, Button, Paw } from "@leadcat/ui"

import { CatStage } from "~/features/landing/components/cat-stage"
import { FloatingMotifs } from "~/features/landing/components/floating-motifs"

export function Hero() {
  return (
    <section className="relative overflow-hidden px-6 pt-8 pb-12 md:pt-16">
      <FloatingMotifs />
      <div className="relative mx-auto grid max-w-6xl items-center gap-8 md:grid-cols-2 md:gap-4">
        <div className="text-center md:text-left">
          <Badge tone="sunny" className="mb-5">
            <Paw className="size-3.5" />
            Scheduling, but cuddly
          </Badge>
          <h1 className="text-4xl leading-tight font-bold text-kitty-800 sm:text-5xl lg:text-6xl">
            Meetings your team will{" "}
            <span className="relative whitespace-nowrap text-coral-500">
              actually love
              <span
                className="absolute -bottom-2 left-0 h-3 w-full rounded-full bg-sunny-300/70"
                aria-hidden="true"
              />
            </span>{" "}
            <Paw className="inline-block size-8 align-middle text-coral-400 sm:size-10" />
          </h1>
          <p className="mx-auto mt-6 max-w-md text-lg text-kitty-600 md:mx-0">
            Lead Cat sniffs out the purrfect time across everyone&apos;s
            calendars - no more back-and-forth, no more double-booking. Just
            happy little meetings.
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-4 md:justify-start">
            <Button size="lg">Get started - it&apos;s free</Button>
            <Button variant="secondary" size="lg">
              See how it works
            </Button>
          </div>
          <p className="mt-5 text-sm font-semibold text-kitty-400">
            Works with Google Calendar &amp; Telegram. No credit card, no claws
            out.
          </p>
        </div>

        <div className="relative mx-auto aspect-square w-full max-w-lg">
          <div
            className="absolute inset-x-6 bottom-6 top-10 rounded-full bg-gradient-to-b from-peach-100 to-cream-200"
            aria-hidden="true"
          />
          <div className="relative h-full w-full">
            <CatStage />
          </div>
        </div>
      </div>
    </section>
  )
}
