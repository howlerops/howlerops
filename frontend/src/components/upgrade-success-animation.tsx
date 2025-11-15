/**
 * Upgrade Success Animation Component
 *
 * Celebration animation shown after successful upgrade.
 * Creates a delightful moment with confetti and smooth transitions.
 *
 * Usage:
 * ```tsx
 * <UpgradeSuccessAnimation
 *   tier="individual"
 *   onComplete={() => handleComplete()}
 * />
 * ```
 */

import * as React from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Sparkles, Check, Crown, Zap, Star } from "lucide-react";
import { cn } from "@/lib/utils";
import { GradientFeatureBadge } from "./feature-badge";
import type { TierLevel } from "@/types/tiers";

interface UpgradeSuccessAnimationProps {
  tier: TierLevel;
  duration?: number;
  onComplete?: () => void;
  className?: string;
}

const tierConfig = {
  local: {
    title: "Welcome!",
    subtitle: "Enjoy Howlerops",
    icon: Check,
    gradient: "from-gray-500 to-gray-600",
    emoji: "✨",
  },
  individual: {
    title: "Welcome to Pro!",
    subtitle: "All features unlocked",
    icon: Sparkles,
    gradient: "from-purple-500 to-pink-500",
    emoji: "🎉",
  },
  team: {
    title: "Welcome to Team!",
    subtitle: "Your team is ready to collaborate",
    icon: Crown,
    gradient: "from-blue-500 to-cyan-500",
    emoji: "🚀",
  },
};

// Confetti particle component
function ConfettiParticle({ delay = 0 }: { delay?: number }) {
  const colors = ["#8b5cf6", "#ec4899", "#3b82f6", "#10b981", "#f59e0b"];
  const randomColor = colors[Math.floor(Math.random() * colors.length)];
  const randomX = Math.random() * 100 - 50;
  const randomRotation = Math.random() * 360;
  const randomDuration = 2 + Math.random() * 2;

  return (
    <motion.div
      className="absolute top-1/2 left-1/2 w-2 h-2 rounded-full"
      style={{ backgroundColor: randomColor }}
      initial={{ opacity: 1, scale: 1, x: 0, y: 0, rotate: 0 }}
      animate={{
        opacity: 0,
        scale: 0.5,
        x: randomX * 10,
        y: Math.random() * 500 - 250,
        rotate: randomRotation,
      }}
      transition={{
        duration: randomDuration,
        delay,
        ease: "easeOut",
      }}
    />
  );
}

// Star burst component
function StarBurst({ delay = 0 }: { delay?: number }) {
  const randomAngle = Math.random() * 360;
  const randomDistance = 100 + Math.random() * 200;

  return (
    <motion.div
      className="absolute top-1/2 left-1/2"
      initial={{ opacity: 1, scale: 0, x: 0, y: 0 }}
      animate={{
        opacity: 0,
        scale: 2,
        x: Math.cos(randomAngle) * randomDistance,
        y: Math.sin(randomAngle) * randomDistance,
      }}
      transition={{
        duration: 1.5,
        delay,
        ease: "easeOut",
      }}
    >
      <Star className="h-4 w-4 text-yellow-400 fill-yellow-400" />
    </motion.div>
  );
}

export function UpgradeSuccessAnimation({
  tier,
  duration = 3000,
  onComplete,
  className,
}: UpgradeSuccessAnimationProps) {
  const [isVisible, setIsVisible] = React.useState(true);
  const config = tierConfig[tier];
  const Icon = config.icon;

  React.useEffect(() => {
    const timer = setTimeout(() => {
      setIsVisible(false);
      setTimeout(() => {
        onComplete?.();
      }, 500);
    }, duration);

    return () => clearTimeout(timer);
  }, [duration, onComplete]);

  return (
    <AnimatePresence>
      {isVisible && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className={cn(
            "fixed inset-0 z-50 flex items-center justify-center",
            className
          )}
        >
          {/* Background overlay */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-background/95 backdrop-blur-md"
          />

          {/* Confetti particles */}
          <div className="absolute inset-0 overflow-hidden pointer-events-none">
            {Array.from({ length: 50 }).map((_, i) => (
              <ConfettiParticle key={i} delay={i * 0.02} />
            ))}
            {Array.from({ length: 20 }).map((_, i) => (
              <StarBurst key={`star-${i}`} delay={i * 0.05} />
            ))}
          </div>

          {/* Main content */}
          <motion.div
            initial={{ scale: 0.8, opacity: 0, y: 20 }}
            animate={{ scale: 1, opacity: 1, y: 0 }}
            exit={{ scale: 0.9, opacity: 0, y: -20 }}
            transition={{
              type: "spring",
              stiffness: 200,
              damping: 20,
            }}
            className="relative z-10 text-center space-y-6 px-4"
          >
            {/* Icon with pulse animation */}
            <motion.div
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{
                type: "spring",
                stiffness: 300,
                damping: 15,
                delay: 0.2,
              }}
              className="flex items-center justify-center"
            >
              <div className="relative">
                {/* Glow effect */}
                <motion.div
                  animate={{
                    scale: [1, 1.2, 1],
                    opacity: [0.3, 0.6, 0.3],
                  }}
                  transition={{
                    duration: 2,
                    repeat: Infinity,
                    ease: "easeInOut",
                  }}
                  className={cn(
                    "absolute inset-0 rounded-full blur-2xl bg-gradient-to-br",
                    config.gradient
                  )}
                />

                {/* Main icon */}
                <div
                  className={cn(
                    "relative w-24 h-24 rounded-full bg-gradient-to-br flex items-center justify-center",
                    config.gradient,
                    "shadow-2xl"
                  )}
                >
                  <Icon className="h-12 w-12 text-white" />
                </div>

                {/* Orbiting particles */}
                {Array.from({ length: 3 }).map((_, i) => (
                  <motion.div
                    key={i}
                    className="absolute top-1/2 left-1/2"
                    animate={{
                      rotate: 360,
                    }}
                    transition={{
                      duration: 3,
                      repeat: Infinity,
                      ease: "linear",
                      delay: i * 0.3,
                    }}
                  >
                    <motion.div
                      className="w-3 h-3 rounded-full bg-white shadow-lg"
                      style={{
                        transform: `translate(-50%, -50%) translateX(60px)`,
                      }}
                      animate={{
                        scale: [1, 1.5, 1],
                      }}
                      transition={{
                        duration: 1,
                        repeat: Infinity,
                        ease: "easeInOut",
                      }}
                    />
                  </motion.div>
                ))}
              </div>
            </motion.div>

            {/* Emoji */}
            <motion.div
              initial={{ scale: 0, rotate: -180 }}
              animate={{ scale: 1, rotate: 0 }}
              transition={{
                type: "spring",
                stiffness: 200,
                damping: 15,
                delay: 0.3,
              }}
              className="text-6xl"
            >
              {config.emoji}
            </motion.div>

            {/* Text */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.4 }}
              className="space-y-3"
            >
              <h2 className="text-4xl font-bold bg-gradient-to-r from-purple-600 to-pink-600 bg-clip-text text-transparent">
                {config.title}
              </h2>
              <p className="text-lg text-muted-foreground">{config.subtitle}</p>
            </motion.div>

            {/* Badge */}
            <motion.div
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: 0.6 }}
              className="flex justify-center"
            >
              <GradientFeatureBadge tier={tier} />
            </motion.div>

            {/* Check mark animation */}
            <motion.div
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{
                type: "spring",
                stiffness: 300,
                damping: 15,
                delay: 0.8,
              }}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-green-500/10 text-green-700 dark:text-green-300 border border-green-500/20"
            >
              <motion.div
                initial={{ scale: 0, rotate: -180 }}
                animate={{ scale: 1, rotate: 0 }}
                transition={{ delay: 0.9 }}
              >
                <Check className="h-5 w-5" />
              </motion.div>
              <span className="font-semibold">Upgrade Complete</span>
            </motion.div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

/**
 * Compact Success Toast
 * Smaller success notification
 */
export function UpgradeSuccessToast({
  tier,
  message,
  onClose,
}: {
  tier: TierLevel;
  message?: string;
  onClose?: () => void;
}) {
  const config = tierConfig[tier];
  const Icon = config.icon;

  React.useEffect(() => {
    const timer = setTimeout(() => {
      onClose?.();
    }, 3000);

    return () => clearTimeout(timer);
  }, [onClose]);

  return (
    <motion.div
      initial={{ opacity: 0, y: 50, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: 50, scale: 0.95 }}
      className="fixed bottom-4 right-4 z-50 max-w-sm"
    >
      <div className="relative overflow-hidden rounded-lg border border-border bg-card shadow-2xl">
        {/* Animated background */}
        <motion.div
          className={cn(
            "absolute inset-0 bg-gradient-to-br opacity-10",
            config.gradient
          )}
          animate={{
            scale: [1, 1.1, 1],
            opacity: [0.1, 0.2, 0.1],
          }}
          transition={{
            duration: 2,
            repeat: Infinity,
            ease: "easeInOut",
          }}
        />

        <div className="relative flex items-start gap-3 p-4">
          {/* Icon */}
          <div
            className={cn(
              "flex-shrink-0 w-10 h-10 rounded-full bg-gradient-to-br flex items-center justify-center",
              config.gradient
            )}
          >
            <Icon className="h-5 w-5 text-white" />
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <h3 className="font-semibold mb-1">{config.title}</h3>
            <p className="text-sm text-muted-foreground">
              {message || config.subtitle}
            </p>
          </div>

          {/* Success check */}
          <div className="flex-shrink-0">
            <div className="w-6 h-6 rounded-full bg-green-500 flex items-center justify-center">
              <Check className="h-4 w-4 text-white" />
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );
}

/**
 * Feature Unlock Animation
 * Shows when a specific feature is unlocked
 */
export function FeatureUnlockAnimation({
  featureName,
  icon: IconComponent,
  onComplete,
}: {
  featureName: string;
  icon?: React.ComponentType<{ className?: string }>;
  onComplete?: () => void;
}) {
  const Icon = IconComponent || Zap;

  React.useEffect(() => {
    const timer = setTimeout(() => {
      onComplete?.();
    }, 2000);

    return () => clearTimeout(timer);
  }, [onComplete]);

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.8 }}
      className="fixed top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 z-50"
    >
      <div className="relative">
        {/* Glow effect */}
        <motion.div
          animate={{
            scale: [1, 1.5, 1],
            opacity: [0.3, 0.6, 0.3],
          }}
          transition={{
            duration: 1.5,
            repeat: Infinity,
            ease: "easeInOut",
          }}
          className="absolute inset-0 bg-gradient-to-br from-purple-500 to-pink-500 rounded-lg blur-2xl"
        />

        {/* Content */}
        <div className="relative bg-card border border-border rounded-lg shadow-2xl p-6 min-w-[250px]">
          <motion.div
            initial={{ scale: 0, rotate: -180 }}
            animate={{ scale: 1, rotate: 0 }}
            transition={{
              type: "spring",
              stiffness: 200,
              damping: 15,
            }}
            className="flex flex-col items-center gap-3 text-center"
          >
            <div className="w-12 h-12 rounded-full bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
              <Icon className="h-6 w-6 text-white" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground mb-1">
                Feature Unlocked
              </p>
              <p className="font-semibold">{featureName}</p>
            </div>
            <div className="flex items-center gap-1">
              <Sparkles className="h-4 w-4 text-yellow-500" />
              <Sparkles className="h-3 w-3 text-yellow-500" />
              <Sparkles className="h-4 w-4 text-yellow-500" />
            </div>
          </motion.div>
        </div>
      </div>
    </motion.div>
  );
}
