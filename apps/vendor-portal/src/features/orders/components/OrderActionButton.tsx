import { useRef, useState } from 'react';
import { Camera, ChefHat, Package, Truck } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/components/ui/Button';
import { uploadOrderPhoto } from '@/shared/services/upload-service';
import { photoRequirementCaption } from '@/features/orders/order-actions';
import type { Carrier, ChefOrderAction } from '@/features/orders/order-actions';
import type { OrderStatus } from '@/shared/types';

interface OrderActionButtonProps {
  orderId: string;
  /** The 'advance' branch of getChefOrderAction — the caller renders waiting
   *  captions itself, so this component only ever handles a real transition. */
  action: Extract<ChefOrderAction, { kind: 'advance' }>;
  /** True while this order's status mutation is in flight. */
  isPending: boolean;
  /** Fired once any required photo has uploaded successfully. */
  onAdvance: (nextStatus: OrderStatus, carrier?: Carrier) => void;
}

function actionIcon(nextStatus: OrderStatus) {
  switch (nextStatus) {
    case 'preparing':
      return <ChefHat className="h-4 w-4" />;
    case 'picked_up':
      return <Truck className="h-4 w-4" />;
    default:
      return <Package className="h-4 w-4" />;
  }
}

/**
 * The chef's next status action, with the vendor app's photo gate attached.
 *
 * On a photo-gated step the button opens the file picker instead of firing the
 * transition — `capture="environment"` makes a phone browser open the rear
 * camera directly (live proof the food is genuinely ready), while a desktop
 * browser falls back to the file dialog. The upload must succeed before the
 * status moves, so a cancelled picker or a failed upload leaves the order
 * exactly where it was and the chef just retries.
 */
export function OrderActionButton({
  orderId,
  action,
  isPending,
  onAdvance,
}: OrderActionButtonProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [isUploading, setIsUploading] = useState(false);
  const { photoKind } = action;

  const handleFile = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    // Always clear the input, otherwise re-picking the same file after a failed
    // upload fires no change event and the button looks dead.
    event.target.value = '';
    if (!file || !photoKind) return;

    setIsUploading(true);
    try {
      await uploadOrderPhoto(orderId, photoKind, file);
      onAdvance(action.nextStatus, action.carrier);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : 'Could not upload the photo. Please try again.'
      );
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="flex flex-1 flex-col gap-1.5">
      <Button
        variant={action.variant}
        size="sm"
        className="w-full"
        leftIcon={photoKind ? <Camera className="h-4 w-4" /> : actionIcon(action.nextStatus)}
        isLoading={isUploading || isPending}
        onClick={() =>
          photoKind ? inputRef.current?.click() : onAdvance(action.nextStatus, action.carrier)
        }
      >
        {action.label}
      </Button>
      {photoKind && (
        <>
          <input
            ref={inputRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            capture="environment"
            className="hidden"
            onChange={handleFile}
          />
          <p className="text-center text-xs text-muted-foreground">
            {photoRequirementCaption(photoKind)}
          </p>
        </>
      )}
    </div>
  );
}
