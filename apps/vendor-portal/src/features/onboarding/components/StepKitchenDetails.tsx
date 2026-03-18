import { useState } from 'react';
import { useOnboardingStore } from '@/app/store/onboarding-store';
import { Input, Textarea } from '@/shared/components/ui/Input';
import { Card } from '@/shared/components/ui/Card';
import { Badge } from '@/shared/components/ui/Badge';
import { Button } from '@/shared/components/ui/Button';
import { ChefHat, Plus, X, Check } from 'lucide-react';
import type { KitchenType } from '@/shared/types';

interface Props {
  errors: Record<string, string>;
}

const CUISINES = [
  'South Indian', 'North Indian', 'Bengali', 'Gujarati', 'Rajasthani',
  'Punjabi', 'Mughlai', 'Kerala', 'Hyderabadi', 'Street Food',
  'Chinese', 'Continental', 'Maharashtrian', 'Bihari', 'Chettinad',
  'Awadhi', 'Konkani', 'Andhra',
];

const KITCHEN_TYPES: { value: KitchenType; label: string; desc: string }[] = [
  { value: 'home_kitchen', label: 'Home Kitchen', desc: 'Cooking from your own home' },
  { value: 'cloud_kitchen', label: 'Cloud Kitchen', desc: 'Dedicated kitchen space, delivery only' },
  { value: 'shared_kitchen', label: 'Shared Kitchen', desc: 'Using a shared commercial kitchen' },
];

const EXPERIENCE_OPTIONS = [
  'Less than 1 year',
  '1-3 years',
  '3-5 years',
  '5-10 years',
  '10+ years',
];

const MEALS_OPTIONS = [
  'Up to 10 meals',
  '10-25 meals',
  '25-50 meals',
  '50-100 meals',
  '100+ meals',
];

export function StepKitchenDetails({ errors }: Props) {
  const { data, updateData } = useOnboardingStore();
  const [newSpecialty, setNewSpecialty] = useState('');

  const toggleCuisine = (cuisine: string) => {
    const updated = data.cuisines.includes(cuisine)
      ? data.cuisines.filter((c) => c !== cuisine)
      : [...data.cuisines, cuisine];
    updateData({ cuisines: updated });
  };

  const addSpecialty = () => {
    const trimmed = newSpecialty.trim();
    if (trimmed && !data.specialties.includes(trimmed)) {
      updateData({ specialties: [...data.specialties, trimmed] });
      setNewSpecialty('');
    }
  };

  const removeSpecialty = (specialty: string) => {
    updateData({ specialties: data.specialties.filter((s) => s !== specialty) });
  };

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex items-center gap-2">
          <ChefHat className="h-5 w-5 text-muted-foreground" />
          <h3 className="text-lg font-semibold text-foreground">Kitchen Information</h3>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          Tell customers about your kitchen and what makes your food special.
        </p>

        <div className="mt-6 space-y-4">
          <Input
            label="Kitchen / Business Name"
            placeholder="e.g. Meena's Kitchen, Amma's Tiffin Service"
            value={data.businessName}
            onChange={(e) => updateData({ businessName: e.target.value })}
            error={errors.businessName}
          />

          <Textarea
            label="Description"
            placeholder="Describe your cooking style, what makes your food special, your signature touch..."
            rows={4}
            value={data.description}
            onChange={(e) => updateData({ description: e.target.value })}
            error={errors.description}
            hint="Min 20 characters. This is shown to customers on your profile."
          />

          {/* Kitchen Type */}
          <div>
            <label className="mb-2 block text-sm font-medium text-foreground">Kitchen Type</label>
            <div className="grid gap-3 sm:grid-cols-3">
              {KITCHEN_TYPES.map((type) => (
                <button
                  key={type.value}
                  type="button"
                  onClick={() => updateData({ kitchenType: type.value })}
                  className={`rounded-xl border-2 p-4 text-left transition-all ${
                    data.kitchenType === type.value
                      ? 'border-primary bg-primary/5 shadow-sm'
                      : 'border-border hover:border-primary/30'
                  }`}
                >
                  <p className="text-sm font-semibold text-foreground">{type.label}</p>
                  <p className="mt-1 text-xs text-muted-foreground">{type.desc}</p>
                </button>
              ))}
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1.5 block text-sm font-medium text-foreground">
                Years of Cooking Experience
              </label>
              <select
                value={data.yearsOfExperience}
                onChange={(e) => updateData({ yearsOfExperience: e.target.value })}
                className="w-full rounded-lg border-2 border-input bg-background px-4 py-2.5 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20"
              >
                <option value="">Select experience</option>
                {EXPERIENCE_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>{opt}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1.5 block text-sm font-medium text-foreground">
                Meals You Can Prepare Daily
              </label>
              <select
                value={data.mealsPerDay}
                onChange={(e) => updateData({ mealsPerDay: e.target.value })}
                className="w-full rounded-lg border-2 border-input bg-background px-4 py-2.5 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20"
              >
                <option value="">Select capacity</option>
                {MEALS_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>{opt}</option>
                ))}
              </select>
            </div>
          </div>
        </div>
      </Card>

      {/* Cuisines */}
      <Card>
        <h3 className="text-lg font-semibold text-foreground">Cuisines You Specialize In</h3>
        <p className="mt-1 text-sm text-muted-foreground">Select at least one cuisine type</p>

        <div className="mt-4 flex flex-wrap gap-2">
          {CUISINES.map((cuisine) => {
            const isSelected = data.cuisines.includes(cuisine);
            return (
              <button
                key={cuisine}
                type="button"
                onClick={() => toggleCuisine(cuisine)}
                className={`inline-flex items-center gap-1.5 rounded-full px-4 py-2 text-sm font-medium transition-all ${
                  isSelected
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
                }`}
              >
                {isSelected && <Check className="h-3.5 w-3.5" />}
                {cuisine}
              </button>
            );
          })}
        </div>
        {errors.cuisines && (
          <p className="mt-2 text-sm text-destructive">{errors.cuisines}</p>
        )}
      </Card>

      {/* Specialties */}
      <Card>
        <h3 className="text-lg font-semibold text-foreground">Signature Dishes & Specialties</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          Add your best dishes — these appear as tags on your profile (optional)
        </p>

        <div className="mt-4 flex flex-wrap gap-2">
          {data.specialties.map((specialty) => (
            <Badge key={specialty} variant="brand" size="lg" className="gap-1.5">
              {specialty}
              <button
                type="button"
                onClick={() => removeSpecialty(specialty)}
                className="ml-0.5 rounded-full p-0.5 hover:bg-primary/20"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          ))}
          {data.specialties.length === 0 && (
            <p className="text-sm text-muted-foreground">No specialties added yet</p>
          )}
        </div>

        <div className="mt-4 flex gap-2">
          <Input
            placeholder="e.g. Butter Chicken, Masala Dosa..."
            value={newSpecialty}
            onChange={(e) => setNewSpecialty(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                addSpecialty();
              }
            }}
            className="flex-1"
          />
          <Button
            type="button"
            variant="outline"
            onClick={addSpecialty}
            disabled={!newSpecialty.trim()}
            leftIcon={<Plus className="h-4 w-4" />}
          >
            Add
          </Button>
        </div>
      </Card>
    </div>
  );
}
