"use client";

import { motion } from "motion/react";
import { Heart } from "lucide-react";
import { cn } from "@/lib/utils";

interface AnimatedHeartProps {
  isLiked: boolean;
  size?: number;
  className?: string;
}

export function AnimatedHeart({ isLiked, size = 16, className }: AnimatedHeartProps) {
  return (
    <motion.div
      className="relative inline-flex items-center justify-center"
      animate={isLiked ? "liked" : "unliked"}
      initial={false}
    >
      {/* Main heart icon */}
      <motion.div
        variants={{
          liked: {
            scale: [1, 1.3, 1],
            rotate: [0, -10, 10, -10, 0],
          },
          unliked: {
            scale: 1,
            rotate: 0,
          },
        }}
        transition={{
          duration: 0.5,
          ease: "easeOut",
        }}
      >
        <Heart
          className={cn(className, isLiked && "fill-current")}
          style={{ width: size, height: size }}
        />
      </motion.div>

      {/* Burst particles when liked */}
      {isLiked && (
        <>
          {[...Array(6)].map((_, i) => (
            <motion.div
              key={i}
              className="absolute"
              initial={{ scale: 0, opacity: 1 }}
              animate={{
                scale: [0, 1, 0],
                opacity: [1, 1, 0],
                x: Math.cos((i * Math.PI * 2) / 6) * 20,
                y: Math.sin((i * Math.PI * 2) / 6) * 20,
              }}
              transition={{
                duration: 0.6,
                ease: "easeOut",
              }}
            >
              <div
                className={cn(
                  "h-1 w-1 rounded-full",
                  i % 2 === 0 ? "bg-red-400" : "bg-pink-400"
                )}
              />
            </motion.div>
          ))}
        </>
      )}

      {/* Ring effect */}
      {isLiked && (
        <motion.div
          className="absolute inset-0 rounded-full border-2 border-red-400"
          initial={{ scale: 1, opacity: 0.8 }}
          animate={{
            scale: [1, 1.8],
            opacity: [0.8, 0],
          }}
          transition={{
            duration: 0.5,
            ease: "easeOut",
          }}
        />
      )}
    </motion.div>
  );
}
