import { Button, CatFace, Paw, motion } from "@leadcat/ui";

export function Footer() {
  return (
    <footer className="relative px-6 pb-12 pt-16">
      <div className="mx-auto max-w-3xl text-center">
        <motion.div
          initial={{ scale: 0.7, opacity: 0 }}
          whileInView={{ scale: 1, opacity: 1 }}
          viewport={{ once: true }}
          transition={{ type: "spring", stiffness: 240, damping: 14 }}
          className="mx-auto mb-6 w-fit"
        >
          <CatFace className="h-24 w-24 animate-wiggle" />
        </motion.div>
        <h2 className="text-3xl font-bold text-kitty-800 sm:text-4xl">
          Ready to make meetings purr?
        </h2>
        <p className="mx-auto mt-3 max-w-md text-kitty-600">
          Join the teams that traded scheduling chaos for a happy little cat.
        </p>
        <div className="mt-7 flex justify-center">
          <Button size="lg">
            Adopt Lead Cat
            <Paw className="h-5 w-5" />
          </Button>
        </div>

        <div className="mt-14 flex flex-col items-center gap-3 text-sm text-kitty-400">
          <div className="flex items-center gap-2 font-bold text-kitty-600">
            <Paw className="h-5 w-5 text-coral-400" />
            Lead Cat
          </div>
          <p>© {new Date().getFullYear()} Lead Cat. Made with whiskers & warmth.</p>
        </div>
      </div>
    </footer>
  );
}
