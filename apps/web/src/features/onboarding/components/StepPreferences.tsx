import { useOnboardingStore } from '@/app/store/onboarding-store';
import { Card } from '@/shared/components/ui/Card';
import { Badge } from '@/shared/components/ui/Badge';
import { usePreferences } from '@/shared/hooks/usePreferences';
import { UtensilsCrossed, Flame } from 'lucide-react';
import { cn } from '@/shared/utils/cn';

export function StepPreferences() {
  const { data, updateData } = useOnboardingStore();
  const { dietary, allergy, cuisine, spiceLevel, householdSize } = usePreferences();

  const toggleArrayItem = (field: 'dietaryPreferences' | 'foodAllergies' | 'cuisinePreferences', value: string) => {
    const current = data[field];
    const updated = current.includes(value)
      ? current.filter((v) => v !== value)
      : [...current, value];
    updateData({ [field]: updated });
  };

  return (
    <div className="space-y-6">
      {/* Dietary Preferences */}
      <Card>
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <UtensilsCrossed className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-foreground">Dietary Preferences</h2>
            <p className="text-sm text-muted-foreground">What type of food do you prefer?</p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {dietary.map((option) => {
            const selected = data.dietaryPreferences.includes(option.value);
            return (
              <button
                key={option.value}
                type="button"
                onClick={() => toggleArrayItem('dietaryPreferences', option.value)}
              >
                <Badge
                  variant={selected ? 'default' : 'outline'}
                  size="lg"
                  className="cursor-pointer"
                >
                  {option.label}
                </Badge>
              </button>
            );
          })}
        </div>
      </Card>

      {/* Food Allergies */}
      <Card>
        <h3 className="font-semibold text-foreground mb-1">Food Allergies</h3>
        <p className="text-sm text-muted-foreground mb-4">Select any allergies so we can flag dishes for you</p>
        <div className="flex flex-wrap gap-2">
          {allergy.map((option) => {
            const selected = data.foodAllergies.includes(option.value);
            return (
              <button
                key={option.value}
                type="button"
                onClick={() => toggleArrayItem('foodAllergies', option.value)}
              >
                <Badge
                  variant={selected ? 'error' : 'outline'}
                  size="lg"
                  className="cursor-pointer"
                >
                  {option.label}
                </Badge>
              </button>
            );
          })}
        </div>
      </Card>

      {/* Cuisine Preferences */}
      <Card>
        <h3 className="font-semibold text-foreground mb-1">Favourite Cuisines</h3>
        <p className="text-sm text-muted-foreground mb-4">Pick cuisines you love - we'll prioritise them in your feed</p>
        <div className="flex flex-wrap gap-2">
          {cuisine.map((option) => {
            const selected = data.cuisinePreferences.includes(option.value);
            return (
              <button
                key={option.value}
                type="button"
                onClick={() => toggleArrayItem('cuisinePreferences', option.value)}
              >
                <Badge
                  variant={selected ? 'brand' : 'outline'}
                  size="lg"
                  className="cursor-pointer"
                >
                  {option.label}
                </Badge>
              </button>
            );
          })}
        </div>
      </Card>

      {/* Spice Tolerance */}
      <Card>
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <Flame className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold text-foreground">Spice Tolerance</h3>
            <p className="text-sm text-muted-foreground">How spicy do you like your food?</p>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {spiceLevel.map((level) => {
            const selected = data.spiceTolerance === level.value;
            return (
              <button
                key={level.value}
                type="button"
                onClick={() => updateData({ spiceTolerance: level.value })}
                className={cn(
                  'rounded-xl border-2 p-3 text-center transition-all',
                  selected
                    ? 'border-primary bg-primary/5 text-primary'
                    : 'border-border hover:border-primary/30'
                )}
              >
                <p className="font-medium">{level.label}</p>
                {level.description && (
                  <p className="text-xs text-muted-foreground mt-0.5">{level.description}</p>
                )}
              </button>
            );
          })}
        </div>
      </Card>

      {/* Household Size */}
      <Card>
        <h3 className="font-semibold text-foreground mb-1">Household Size</h3>
        <p className="text-sm text-muted-foreground mb-4">Helps us suggest the right portion sizes</p>
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-5">
          {householdSize.map((size) => {
            const selected = data.householdSize === size.value;
            return (
              <button
                key={size.value}
                type="button"
                onClick={() => updateData({ householdSize: size.value })}
                className={cn(
                  'rounded-xl border-2 p-3 text-center transition-all',
                  selected
                    ? 'border-primary bg-primary/5 text-primary'
                    : 'border-border hover:border-primary/30'
                )}
              >
                <p className="text-sm font-medium">{size.label}</p>
              </button>
            );
          })}
        </div>
      </Card>
    </div>
  );
}
