import { Bike } from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';
import type { Carrier, CarrierSwitch } from '@/features/orders/order-actions';

interface CarrierSwitchButtonProps {
  switchTo: CarrierSwitch;
  disabled: boolean;
  onSwitch: (carrier: Carrier) => void;
}

/**
 * The carrier escape on a live order — "I'll deliver this instead" / "Hand to a
 * rider instead". Quiet by default so the primary status action stays the hero,
 * but promoted to a full button the moment it carries a hint, because a hint
 * means it is the ONLY remaining route to a completed order.
 */
export function CarrierSwitchButton({ switchTo, disabled, onSwitch }: CarrierSwitchButtonProps) {
  const isOnlyRoute = Boolean(switchTo.hint);

  return (
    <div className="flex flex-col gap-1">
      {switchTo.hint && (
        <p className="text-center text-xs text-muted-foreground">{switchTo.hint}</p>
      )}
      <Button
        variant={isOnlyRoute ? 'brand-outline' : 'ghost'}
        size="sm"
        className="w-full"
        leftIcon={<Bike className="h-4 w-4" />}
        disabled={disabled}
        onClick={() => onSwitch(switchTo.carrier)}
      >
        {switchTo.label}
      </Button>
    </div>
  );
}
