import { Input } from '@/shared/components/ui/Input';
import { useOnboardingStore } from '@/app/store/onboarding-store';
import { Card } from '@/shared/components/ui/Card';
import { User } from 'lucide-react';

interface StepBasicInfoProps {
  errors: Record<string, string>;
}

export function StepBasicInfo({ errors }: StepBasicInfoProps) {
  const { data, updateData } = useOnboardingStore();

  return (
    <Card>
      <div className="flex items-center gap-3 mb-6">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          <User className="h-5 w-5 text-primary" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-foreground">Basic Information</h2>
          <p className="text-sm text-muted-foreground">Tell us a bit about yourself</p>
        </div>
      </div>

      <div className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <Input
            label="First name *"
            value={data.firstName}
            onChange={(e) => updateData({ firstName: e.target.value })}
            error={errors.firstName}
            placeholder="Enter your first name"
          />
          <Input
            label="Last name *"
            value={data.lastName}
            onChange={(e) => updateData({ lastName: e.target.value })}
            error={errors.lastName}
            placeholder="Enter your last name"
          />
        </div>

        <Input
          label="Phone number"
          type="tel"
          value={data.phone}
          onChange={(e) => updateData({ phone: e.target.value })}
          placeholder="+91 98765 43210"
        />

        <Input
          label="Date of birth"
          type="date"
          value={data.dateOfBirth}
          onChange={(e) => updateData({ dateOfBirth: e.target.value })}
        />
      </div>
    </Card>
  );
}
