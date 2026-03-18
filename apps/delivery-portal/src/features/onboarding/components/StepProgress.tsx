import { Check } from 'lucide-react';

const steps = [
  { number: 1, label: 'Personal Info' },
  { number: 2, label: 'Vehicle Details' },
  { number: 3, label: 'Documents' },
  { number: 4, label: 'Plan' },
  { number: 5, label: 'Review' },
];

interface StepProgressProps {
  currentStep: number;
}

export function StepProgress({ currentStep }: StepProgressProps) {
  return (
    <div className="flex items-center justify-between">
      {steps.map((step, index) => {
        const isCompleted = currentStep > step.number;
        const isCurrent = currentStep === step.number;

        return (
          <div key={step.number} className="flex items-center flex-1 last:flex-none">
            <div className="flex flex-col items-center">
              <div
                className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-semibold transition-all ${
                  isCompleted
                    ? 'bg-primary text-primary-foreground'
                    : isCurrent
                      ? 'border-2 border-primary bg-primary/10 text-primary'
                      : 'border-2 border-border bg-background text-muted-foreground'
                }`}
              >
                {isCompleted ? <Check className="h-4 w-4" /> : step.number}
              </div>
              <span
                className={`mt-1.5 text-[10px] font-medium whitespace-nowrap ${
                  isCurrent ? 'text-primary' : isCompleted ? 'text-foreground' : 'text-muted-foreground'
                }`}
              >
                {step.label}
              </span>
            </div>
            {index < steps.length - 1 && (
              <div
                className={`mx-1 h-0.5 flex-1 rounded-full transition-all ${
                  currentStep > step.number ? 'bg-primary' : 'bg-border'
                }`}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}
