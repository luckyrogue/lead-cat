import { Paw, motion } from "@leadcat/ui";

export function Nav() {
  return (
    <motion.nav
      initial={{ opacity: 0, y: -16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="mx-auto flex max-w-6xl items-center justify-between px-6 py-6"
    >
      <div className="flex items-center gap-2 text-xl font-bold text-kitty-800">
        <span className="grid h-9 w-9 place-items-center rounded-full bg-coral-500 text-white shadow-pop">
          <Paw className="h-5 w-5" />
        </span>
        Lead Cat
      </div>
      <a
        href="#"
        className="rounded-full px-5 py-2 text-sm font-bold text-coral-600 transition-colors hover:bg-coral-50"
      >
        Sign in
      </a>
    </motion.nav>
  );
}
