import { Camera } from 'lucide-react';

interface OrderLifecyclePhotosProps {
  orderNumber: string;
  readyPhotoUrl?: string;
  handoverPhotoUrl?: string;
  /** Compact variant for the dense history list. */
  size?: 'sm' | 'md';
}

/**
 * The lifecycle photos already attached to an order — the chef's own proof that
 * the upload landed, and the record they can point at in a dispute. Renders
 * nothing until at least one photo exists, so pre-photo orders (and every order
 * still in prep) stay visually unchanged.
 */
export function OrderLifecyclePhotos({
  orderNumber,
  readyPhotoUrl,
  handoverPhotoUrl,
  size = 'md',
}: OrderLifecyclePhotosProps) {
  const photos = [
    { url: readyPhotoUrl, label: 'Food ready' },
    { url: handoverPhotoUrl, label: 'Handover proof' },
  ].filter((p): p is { url: string; label: string } => Boolean(p.url));

  if (photos.length === 0) return null;

  const box = size === 'sm' ? 'h-14 w-14' : 'h-20 w-20';

  return (
    <div className="flex items-center gap-3">
      <Camera aria-hidden="true" className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      {photos.map((photo) => (
        <a
          key={photo.label}
          href={photo.url}
          target="_blank"
          rel="noopener noreferrer"
          className="group flex items-center gap-2 rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        >
          <img
            src={photo.url}
            alt={`${photo.label} photo for order ${orderNumber}`}
            loading="lazy"
            className={`${box} rounded-lg border border-border object-cover transition-opacity duration-200 group-hover:opacity-90`}
          />
          <span className="text-xs text-muted-foreground group-hover:text-foreground">
            {photo.label}
          </span>
        </a>
      ))}
    </div>
  );
}
