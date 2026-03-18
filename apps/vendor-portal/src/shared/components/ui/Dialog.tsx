import { forwardRef } from 'react';
import {
  Dialog as DSDialog,
  DialogClose as DSDialogClose,
  DialogTrigger as DSDialogTrigger,
  DialogContent as DSDialogContent,
  DialogHeader as DSDialogHeader,
  DialogFooter as DSDialogFooter,
  DialogTitle as DSDialogTitle,
  DialogDescription as DSDialogDescription,
} from '@tesserix/web';
import { cn } from '@tesserix/web';
import * as DialogPrimitive from '@radix-ui/react-dialog';

// Re-export DS Dialog components — these all share the same DS DialogContext
const Dialog = DSDialog;
const DialogTrigger = DSDialogTrigger;
const DialogClose = DSDialogClose;
const DialogContent = DSDialogContent;
const DialogHeader = DSDialogHeader;
const DialogFooter = DSDialogFooter;
const DialogTitle = DSDialogTitle;
const DialogDescription = DSDialogDescription;

// ---------------------------------------------------------------------------
// Alert Dialog — built on Radix primitives (separate context from DS Dialog)
// ---------------------------------------------------------------------------

const AlertDialog = DialogPrimitive.Root;
const AlertDialogTrigger = DialogPrimitive.Trigger;

const AlertDialogOverlay = forwardRef<
  React.ElementRef<typeof DialogPrimitive.Overlay>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Overlay>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Overlay
    ref={ref}
    className={cn(
      'fixed inset-0 z-50 bg-background/80 backdrop-blur-sm',
      'data-[state=open]:animate-in data-[state=closed]:animate-out',
      'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
      className
    )}
    {...props}
  />
));
AlertDialogOverlay.displayName = 'AlertDialogOverlay';

interface AlertDialogContentProps
  extends React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content> {
  variant?: 'default' | 'danger';
}

const AlertDialogContent = forwardRef<
  React.ElementRef<typeof DialogPrimitive.Content>,
  AlertDialogContentProps
>(({ className, children, variant: _variant = 'default', ...props }, ref) => (
  <DialogPrimitive.Portal>
    <AlertDialogOverlay />
    <DialogPrimitive.Content
      ref={ref}
      className={cn(
        'fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2',
        'w-full max-w-md p-6',
        'border bg-card text-card-foreground shadow-modal rounded-2xl',
        'data-[state=open]:animate-in data-[state=closed]:animate-out',
        'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
        'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
        className
      )}
      {...props}
    >
      {children}
    </DialogPrimitive.Content>
  </DialogPrimitive.Portal>
));
AlertDialogContent.displayName = 'AlertDialogContent';

const AlertDialogHeader = DialogHeader;
const AlertDialogFooter = DialogFooter;
const AlertDialogTitle = DSDialogTitle;
const AlertDialogDescription = DSDialogDescription;
const AlertDialogCancel = DialogPrimitive.Close;

const AlertDialogAction = forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement>
>(({ className, ...props }, ref) => (
  <button
    ref={ref}
    className={cn(
      'inline-flex h-10 items-center justify-center rounded-lg bg-primary px-4 font-medium text-primary-foreground',
      'hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
      'disabled:opacity-50 disabled:pointer-events-none',
      className
    )}
    {...props}
  />
));
AlertDialogAction.displayName = 'AlertDialogAction';

// ---------------------------------------------------------------------------
// SimpleDialog — convenience wrapper using DS Dialog
// ---------------------------------------------------------------------------

interface SimpleDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title?: string;
  description?: string;
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
  children: React.ReactNode;
}

const SimpleDialog = ({
  open,
  onOpenChange,
  title,
  description,
  size = 'md',
  children,
}: SimpleDialogProps) => (
  <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent size={size}>
      {(title || description) && (
        <DialogHeader>
          {title && <DialogTitle>{title}</DialogTitle>}
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>
      )}
      {children}
      <DialogClose />
    </DialogContent>
  </Dialog>
);

export {
  Dialog,
  DialogClose,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
  SimpleDialog,
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogCancel,
  AlertDialogAction,
};
