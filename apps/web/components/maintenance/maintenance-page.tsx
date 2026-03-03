'use client';

import { motion } from 'motion/react';

export function MaintenancePage() {
  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background">
      {/* Subtle Background Gradients to match theme */}
      <div
        aria-hidden
        className="absolute inset-0 isolate hidden opacity-40 contain-strict lg:block"
      >
        <div className="w-140 h-320 -translate-y-87.5 absolute left-0 top-0 -rotate-45 rounded-full bg-[radial-gradient(68.54%_68.72%_at_55.02%_31.46%,hsla(0,0%,85%,.08)_0,hsla(0,0%,55%,.02)_50%,hsla(0,0%,45%,0)_80%)]" />
        <div className="h-320 absolute left-0 top-0 w-60 -rotate-45 rounded-full bg-[radial-gradient(50%_50%_at_50%_50%,hsla(0,0%,85%,.06)_0,hsla(0,0%,45%,.02)_80%,transparent_100%)] [translate:5%_-50%]" />
      </div>

      <div className="relative z-10 mx-auto max-w-4xl px-6">
        <div className="flex flex-col items-center text-center">
          <motion.h1
            initial={{ opacity: 0, filter: 'blur(12px)', y: 20 }}
            animate={{ opacity: 1, filter: 'blur(0px)', y: 0 }}
            transition={{
              type: 'spring',
              bounce: 0.3,
              duration: 1.5,
            }}
            className="text-balance text-2xl font-medium tracking-tight sm:text-3xl md:text-4xl text-foreground"
          >
            Bentar ya, lagi bersih-bersih bug.
          </motion.h1>
          
          <motion.div
            initial={{ scaleX: 0, opacity: 0 }}
            animate={{ scaleX: 1, opacity: 1 }}
            transition={{ delay: 0.8, duration: 1, ease: 'easeInOut' }}
            className="mt-8 h-px w-16 bg-primary/50"
          />
        </div>
      </div>
    </main>
  );
}
