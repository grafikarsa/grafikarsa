'use client';

import { Check } from 'lucide-react';
import { cn } from '@/lib/utils';

interface WizardProgressProps {
  currentStep: number;
  totalSteps: number;
  steps: { label: string; description: string }[];
}

export function WizardProgress({ currentStep, totalSteps, steps }: WizardProgressProps) {
  return (
    <div className="w-full">
      {/* Progress Bar */}
      <div className="mb-8 flex items-start justify-between">
        {steps.map((step, index) => {
          const stepNumber = index + 1;
          const isCompleted = stepNumber < currentStep;
          const isCurrent = stepNumber === currentStep;
          const isUpcoming = stepNumber > currentStep;

          return (
            <div key={stepNumber} className="flex flex-1 items-start">
              {/* Step Circle and Label */}
              <div className="flex w-full flex-col items-center">
                <div
                  className={cn(
                    'flex h-10 w-10 shrink-0 items-center justify-center rounded-full border-2 font-semibold transition-all',
                    isCompleted && 'border-primary bg-primary text-primary-foreground',
                    isCurrent && 'border-primary bg-primary/10 text-primary',
                    isUpcoming && 'border-muted-foreground/30 bg-background text-muted-foreground'
                  )}
                >
                  {isCompleted ? (
                    <Check className="h-5 w-5" />
                  ) : (
                    <span className="text-sm">{stepNumber}</span>
                  )}
                </div>
                <div className="mt-2 hidden w-full text-center md:block">
                  <p
                    className={cn(
                      'text-sm font-medium',
                      (isCompleted || isCurrent) && 'text-foreground',
                      isUpcoming && 'text-muted-foreground'
                    )}
                  >
                    {step.label}
                  </p>
                  <p className="mt-0.5 text-xs text-muted-foreground">{step.description}</p>
                </div>
              </div>

              {/* Connector Line */}
              {index < totalSteps - 1 && (
                <div className="flex w-full items-center px-2 pt-5">
                  <div
                    className={cn(
                      'h-0.5 w-full transition-all',
                      stepNumber < currentStep ? 'bg-primary' : 'bg-muted-foreground/30'
                    )}
                  />
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Mobile Step Info */}
      <div className="mb-4 text-center md:hidden">
        <p className="text-sm font-medium">{steps[currentStep - 1].label}</p>
        <p className="text-xs text-muted-foreground">{steps[currentStep - 1].description}</p>
      </div>
    </div>
  );
}
