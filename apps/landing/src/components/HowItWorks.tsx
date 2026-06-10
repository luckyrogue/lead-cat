import { Tag, motion } from "@leadcat/ui";

const steps = [
  {
    n: "1",
    title: "Connect your calendars",
    body: "Link Google in two taps. Lead Cat learns when you're free.",
  },
  {
    n: "2",
    title: "Share your kitty link",
    body: "Drop one link in chat. Guests pick a slot that just works.",
  },
  {
    n: "3",
    title: "Show up & relax",
    body: "Auto-invites, reminders and reschedules. You do nothing.",
  },
];

const stats = [
  { value: "3 sec", label: "to book a meeting" },
  { value: "0", label: "double-bookings" },
  { value: "100%", label: "more purring" },
];

export function HowItWorks() {
  return (
    <section className="relative px-6 py-12">
      <div className="mx-auto max-w-5xl rounded-4xl bg-gradient-to-br from-coral-400 to-peach-400 p-8 shadow-soft sm:p-12">
        <div className="mb-10 text-center">
          <Tag className="border-white/40 bg-white/20 text-white">How it works</Tag>
          <h2 className="mt-4 text-3xl font-bold text-white sm:text-4xl">
            Three little steps, zero headaches
          </h2>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          {steps.map((s, i) => (
            <motion.div
              key={s.n}
              initial={{ opacity: 0, y: 24 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: i * 0.12, type: "spring", stiffness: 220, damping: 18 }}
              className="rounded-3xl bg-white/90 p-6 text-center backdrop-blur"
            >
              <div className="mx-auto mb-4 grid h-12 w-12 place-items-center rounded-full bg-sunny-400 text-xl font-bold text-kitty-800 shadow-pop-sunny">
                {s.n}
              </div>
              <h3 className="text-lg font-bold text-kitty-800">{s.title}</h3>
              <p className="mt-2 text-sm text-kitty-600">{s.body}</p>
            </motion.div>
          ))}
        </div>

        <div className="mt-10 grid grid-cols-3 gap-4 border-t border-white/30 pt-8 text-center text-white">
          {stats.map((st) => (
            <div key={st.label}>
              <div className="text-3xl font-bold sm:text-4xl">{st.value}</div>
              <div className="mt-1 text-xs font-semibold uppercase tracking-wider text-white/80">
                {st.label}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
